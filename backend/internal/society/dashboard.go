package society

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PlatformDashboardSummary models high-level ecosystem health metrics.
type PlatformDashboardSummary struct {
	Societies              PlatformSocietiesMetric `json:"societies"`
	NewSocieties30d        int                     `json:"new_societies_30d"`
	TotalRegisteredUnits   int                     `json:"total_registered_units"`
	Members                PlatformMembersMetric   `json:"members"`
	FinancialVelocity30d   FinancialAmountMetric   `json:"financial_velocity_30d"`
	PlatformAuditEvents24h int                     `json:"platform_audit_events_24h"`
}

type PlatformSocietiesMetric struct {
	Total     int `json:"total"`
	Active    int `json:"active"`
	Pending   int `json:"pending"`
	Suspended int `json:"suspended"`
}

type PlatformMembersMetric struct {
	TotalActiveAccounts int `json:"total_active_accounts"`
}

type FinancialAmountMetric struct {
	AmountPaise  int64  `json:"amount_paise"`
	FormattedINR string `json:"formatted_inr"`
}

// SocietyDashboardSummary models operational, occupancy, gate, and financial scorecards for a specific society.
type SocietyDashboardSummary struct {
	SocietyID     uuid.UUID                  `json:"society_id"`
	Occupancy     SocietyOccupancyMetric     `json:"occupancy"`
	Members       SocietyMembersMetric       `json:"members"`
	Gate          SocietyGateMetric          `json:"gate"`
	Finance       SocietyFinanceMetric       `json:"finance"`
	Communication SocietyCommunicationMetric `json:"communication"`
}

type SocietyOccupancyMetric struct {
	TotalUnits           int     `json:"total_units"`
	OccupiedUnits        int     `json:"occupied_units"`
	VacantUnits          int     `json:"vacant_units"`
	OccupancyRatePercent float64 `json:"occupancy_rate_percent"`
	OwnerOccupiedUnits   int     `json:"owner_occupied_units"`
	TenantOccupiedUnits  int     `json:"tenant_occupied_units"`
}

type SocietyMembersMetric struct {
	TotalActiveMembers int `json:"total_active_members"`
}

type SocietyGateMetric struct {
	VisitorsOnPremises int `json:"visitors_on_premises"`
	TodayEntries       int `json:"today_entries"`
	TodayExits         int `json:"today_exits"`
	PendingApprovals   int `json:"pending_approvals"`
}

type SocietyFinanceMetric struct {
	CurrentPeriodID             *uuid.UUID `json:"current_period_id,omitempty"`
	CurrentPeriodName           string     `json:"current_period_name"`
	BilledPaise                 int64      `json:"billed_paise"`
	CollectedPaise              int64      `json:"collected_paise"`
	CollectionEfficiencyPercent float64    `json:"collection_efficiency_percent"`
	OverdueArrearsPaise         int64      `json:"overdue_arrears_paise"`
	PaymentsCollectedMTDPaise   int64      `json:"payments_collected_mtd_paise"`
}

type SocietyCommunicationMetric struct {
	ActiveAnnouncementsCount int `json:"active_announcements_count"`
}

// GateDashboardSummary models the immediate operational shift metrics for gatekeepers and security.
type GateDashboardSummary struct {
	SocietyID          uuid.UUID `json:"society_id"`
	VisitorsOnPremises int       `json:"visitors_on_premises"`
	TodayEntries       int       `json:"today_entries"`
	TodayExits         int       `json:"today_exits"`
	PendingApprovals   int       `json:"pending_approvals"`
	ActiveGuardsOnDuty int       `json:"active_guards_on_duty"`
}

// FormatINR formats an integer paise count into standard Indian currency formatting (e.g. ₹4,85,000.00).
func FormatINR(paise int64) string {
	isNegative := paise < 0
	absPaise := paise
	if isNegative {
		absPaise = -paise
	}

	rupees := absPaise / 100
	remainder := absPaise % 100

	rupStr := strconv.FormatInt(rupees, 10)
	n := len(rupStr)

	var formattedRup string
	if n <= 3 {
		formattedRup = rupStr
	} else {
		lastThree := rupStr[n-3:]
		remaining := rupStr[:n-3]

		var groups []string
		for len(remaining) > 2 {
			groups = append([]string{remaining[len(remaining)-2:]}, groups...)
			remaining = remaining[:len(remaining)-2]
		}
		if len(remaining) > 0 {
			groups = append([]string{remaining}, groups...)
		}
		formattedRup = strings.Join(groups, ",") + "," + lastThree
	}

	sign := ""
	if isNegative {
		sign = "-"
	}

	return fmt.Sprintf("%s₹%s.%02d", sign, formattedRup, remainder)
}

// GetPlatformDashboardSummary retrieves ecosystem health metrics across all societies.
func (r *Repository) GetPlatformDashboardSummary(ctx context.Context) (*PlatformDashboardSummary, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	var summary PlatformDashboardSummary

	// 1. Societies breakdown & 30d creation count
	socQuery := `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'ACTIVE'),
			COUNT(*) FILTER (WHERE status = 'PENDING'),
			COUNT(*) FILTER (WHERE status = 'SUSPENDED'),
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '30 days')
		FROM societies
	`
	err := r.pool.QueryRow(ctx, socQuery).Scan(
		&summary.Societies.Total,
		&summary.Societies.Active,
		&summary.Societies.Pending,
		&summary.Societies.Suspended,
		&summary.NewSocieties30d,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate societies metrics: %w", err)
	}

	// 2. Total registered units
	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM society_units`).Scan(&summary.TotalRegisteredUnits)
	if err != nil {
		return nil, fmt.Errorf("failed to count society units: %w", err)
	}

	// 3. Global active member accounts
	memQuery := `
		SELECT COUNT(DISTINCT user_id)
		FROM society_members
		WHERE status = 'ACTIVE' AND user_id IS NOT NULL
	`
	err = r.pool.QueryRow(ctx, memQuery).Scan(&summary.Members.TotalActiveAccounts)
	if err != nil {
		return nil, fmt.Errorf("failed to count global members: %w", err)
	}

	// 4. Financial velocity 30d (cleared/recorded maintenance payments, excluding reversed)
	finQuery := `
		SELECT ROUND(COALESCE(SUM(amount), 0) * 100)::BIGINT
		FROM maintenance_payments
		WHERE status != 'REVERSED'
		  AND payment_date >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date - INTERVAL '30 days'
	`
	err = r.pool.QueryRow(ctx, finQuery).Scan(&summary.FinancialVelocity30d.AmountPaise)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate financial velocity: %w", err)
	}
	summary.FinancialVelocity30d.FormattedINR = FormatINR(summary.FinancialVelocity30d.AmountPaise)

	// 5. Platform audit events in last 24h
	auditQuery := `
		SELECT COUNT(*)
		FROM platform_audit_logs
		WHERE created_at >= NOW() - INTERVAL '24 hours'
	`
	err = r.pool.QueryRow(ctx, auditQuery).Scan(&summary.PlatformAuditEvents24h)
	if err != nil {
		return nil, fmt.Errorf("failed to count platform audit events: %w", err)
	}

	return &summary, nil
}

// GetSocietyDashboardSummary calculates comprehensive operational, occupancy, gate, and financial scorecards for a society.
func (r *Repository) GetSocietyDashboardSummary(ctx context.Context, societyID uuid.UUID) (*SocietyDashboardSummary, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	summary := SocietyDashboardSummary{
		SocietyID: societyID,
	}

	// 1. Units total & active count
	unitQuery := `
		SELECT COUNT(*)
		FROM society_units
		WHERE society_id = $1 AND status = 'ACTIVE'
	`
	err := r.pool.QueryRow(ctx, unitQuery, societyID).Scan(&summary.Occupancy.TotalUnits)
	if err != nil {
		return nil, fmt.Errorf("failed to count society units: %w", err)
	}

	// 2. Unit occupancy based on active occupant relationships
	occQuery := `
		WITH active_occupant_rels AS (
			SELECT DISTINCT ON (unit_id) unit_id, relationship_type
			FROM unit_relationships
			WHERE society_id = $1 AND status = 'ACTIVE' AND is_occupant = true
		)
		SELECT 
			COUNT(DISTINCT aor.unit_id) AS occupied_units,
			COUNT(DISTINCT aor.unit_id) FILTER (WHERE aor.relationship_type IN ('PRIMARY_OWNER', 'SECONDARY_OWNER', 'OWNER_FAMILY')) AS owner_occupied,
			COUNT(DISTINCT aor.unit_id) FILTER (WHERE aor.relationship_type IN ('PRIMARY_TENANT', 'SECONDARY_TENANT', 'TENANT_FAMILY', 'AUTHORIZED_OCCUPANT')) AS tenant_occupied
		FROM active_occupant_rels aor
	`
	err = r.pool.QueryRow(ctx, occQuery, societyID).Scan(
		&summary.Occupancy.OccupiedUnits,
		&summary.Occupancy.OwnerOccupiedUnits,
		&summary.Occupancy.TenantOccupiedUnits,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate occupancy metrics: %w", err)
	}

	summary.Occupancy.VacantUnits = summary.Occupancy.TotalUnits - summary.Occupancy.OccupiedUnits
	if summary.Occupancy.VacantUnits < 0 {
		summary.Occupancy.VacantUnits = 0
	}
	if summary.Occupancy.TotalUnits > 0 {
		summary.Occupancy.OccupancyRatePercent = math.Round((float64(summary.Occupancy.OccupiedUnits)*100.0/float64(summary.Occupancy.TotalUnits))*10) / 10
	}

	// 3. Active members count
	memQuery := `
		SELECT COUNT(*)
		FROM society_members
		WHERE society_id = $1 AND status = 'ACTIVE'
	`
	err = r.pool.QueryRow(ctx, memQuery, societyID).Scan(&summary.Members.TotalActiveMembers)
	if err != nil {
		return nil, fmt.Errorf("failed to count active members: %w", err)
	}

	// 4. Gate metrics
	gateQuery := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'CHECKED_IN') AS visitors_on_premises,
			COUNT(*) FILTER (WHERE check_in_at >= ((CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date AT TIME ZONE 'Asia/Kolkata')) AS today_entries,
			COUNT(*) FILTER (WHERE check_out_at >= ((CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date AT TIME ZONE 'Asia/Kolkata')) AS today_exits,
			COUNT(*) FILTER (WHERE status = 'PENDING_APPROVAL') AS pending_approvals
		FROM society_visits
		WHERE society_id = $1
	`
	err = r.pool.QueryRow(ctx, gateQuery, societyID).Scan(
		&summary.Gate.VisitorsOnPremises,
		&summary.Gate.TodayEntries,
		&summary.Gate.TodayExits,
		&summary.Gate.PendingApprovals,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate gate metrics: %w", err)
	}

	// 5. Active announcements count
	annQuery := `
		SELECT COUNT(*)
		FROM society_announcements
		WHERE society_id = $1 AND status = 'PUBLISHED'
	`
	err = r.pool.QueryRow(ctx, annQuery, societyID).Scan(&summary.Communication.ActiveAnnouncementsCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count active announcements: %w", err)
	}

	// 6. Finance: find current active billing period
	periodQuery := `
		SELECT id, period_name
		FROM maintenance_billing_periods
		WHERE society_id = $1
		ORDER BY 
			CASE WHEN (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date BETWEEN period_start AND period_end THEN 0 ELSE 1 END,
			period_start DESC
		LIMIT 1
	`
	var periodID uuid.UUID
	var periodName string
	err = r.pool.QueryRow(ctx, periodQuery, societyID).Scan(&periodID, &periodName)
	if err == nil {
		summary.Finance.CurrentPeriodID = &periodID
		summary.Finance.CurrentPeriodName = periodName

		// Billed amount for current period
		billedQuery := `
			SELECT ROUND(COALESCE(SUM(total_amount), 0) * 100)::BIGINT
			FROM maintenance_bills
			WHERE society_id = $1 AND billing_period_id = $2 AND status = 'ISSUED'
		`
		_ = r.pool.QueryRow(ctx, billedQuery, societyID, periodID).Scan(&summary.Finance.BilledPaise)

		// Collected amount for current period (active allocations only)
		collectedQuery := `
			SELECT ROUND(COALESCE(SUM(alloc.allocated_amount), 0) * 100)::BIGINT
			FROM maintenance_payment_allocations alloc
			JOIN maintenance_bills b ON alloc.bill_id = b.id AND alloc.society_id = b.society_id
			WHERE b.society_id = $1 AND b.billing_period_id = $2 AND alloc.status = 'ACTIVE' AND b.status = 'ISSUED'
		`
		_ = r.pool.QueryRow(ctx, collectedQuery, societyID, periodID).Scan(&summary.Finance.CollectedPaise)

		if summary.Finance.BilledPaise > 0 {
			summary.Finance.CollectionEfficiencyPercent = math.Round((float64(summary.Finance.CollectedPaise)*100.0/float64(summary.Finance.BilledPaise))*10) / 10
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to resolve billing period: %w", err)
	}

	// Overdue arrears: all issued bills where due_date < today in IST, minus active allocations
	arrearsQuery := `
		WITH bill_allocs AS (
			SELECT b.id, b.total_amount,
			       COALESCE(SUM(alloc.allocated_amount), 0) AS paid_amount
			FROM maintenance_bills b
			LEFT JOIN maintenance_payment_allocations alloc 
			       ON b.id = alloc.bill_id AND alloc.society_id = b.society_id AND alloc.status = 'ACTIVE'
			WHERE b.society_id = $1 
			  AND b.status = 'ISSUED' 
			  AND b.due_date < (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
			GROUP BY b.id, b.total_amount
		)
		SELECT ROUND(COALESCE(SUM(GREATEST(0.00, total_amount - paid_amount)), 0) * 100)::BIGINT
		FROM bill_allocs
	`
	_ = r.pool.QueryRow(ctx, arrearsQuery, societyID).Scan(&summary.Finance.OverdueArrearsPaise)

	// Payments collected MTD
	mtdQuery := `
		SELECT ROUND(COALESCE(SUM(amount), 0) * 100)::BIGINT
		FROM maintenance_payments
		WHERE society_id = $1 
		  AND status = 'RECORDED' 
		  AND payment_date >= DATE_TRUNC('month', (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date)::date
	`
	_ = r.pool.QueryRow(ctx, mtdQuery, societyID).Scan(&summary.Finance.PaymentsCollectedMTDPaise)

	return &summary, nil
}

// GetGateDashboardSummary calculates real-time shift and perimeter flow metrics for gatekeepers and security.
func (r *Repository) GetGateDashboardSummary(ctx context.Context, societyID uuid.UUID) (*GateDashboardSummary, error) {
	if r.pool == nil {
		return nil, errors.New("database pool unavailable")
	}

	summary := GateDashboardSummary{
		SocietyID: societyID,
	}

	gateQuery := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'CHECKED_IN') AS visitors_on_premises,
			COUNT(*) FILTER (WHERE check_in_at >= ((CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date AT TIME ZONE 'Asia/Kolkata')) AS today_entries,
			COUNT(*) FILTER (WHERE check_out_at >= ((CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date AT TIME ZONE 'Asia/Kolkata')) AS today_exits,
			COUNT(*) FILTER (WHERE status = 'PENDING_APPROVAL') AS pending_approvals
		FROM society_visits
		WHERE society_id = $1
	`
	err := r.pool.QueryRow(ctx, gateQuery, societyID).Scan(
		&summary.VisitorsOnPremises,
		&summary.TodayEntries,
		&summary.TodayExits,
		&summary.PendingApprovals,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate gate visitor metrics: %w", err)
	}

	// Active guards on duty
	guardQuery := `
		SELECT COUNT(DISTINCT user_id)
		FROM society_gate_sessions
		WHERE society_id = $1 AND status = 'ACTIVE' AND ended_at IS NULL
	`
	err = r.pool.QueryRow(ctx, guardQuery, societyID).Scan(&summary.ActiveGuardsOnDuty)
	if err != nil {
		return nil, fmt.Errorf("failed to count active guards: %w", err)
	}

	return &summary, nil
}
