import { describe, it, expect, vi, beforeEach } from 'vitest';
import { societyApi, type Visit } from '$lib/api/society';
import { api } from '$lib/api/client';

describe('Visitor QR and Self-Registration Frontend API', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	describe('Public QR Endpoints', () => {
		it('resolves public registration context by opaque token', async () => {
			const mockContext = {
				society_id: 'soc-123',
				society_name: 'Palm Grove Residency',
				society_code: 'PALM01',
				gate_id: 'gate-456',
				gate_name: 'Main Gate',
				token: 'vqr_token_abc'
			};
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockContext);

			const result = await societyApi.resolvePublicVisitQR('vqr_token_abc');
			expect(result).toEqual(mockContext);
			expect(getSpy).toHaveBeenCalledWith('/public/visit-qr/vqr_token_abc');
		});

		it('dispatches visitor OTP verification code', async () => {
			const mockResp = {
				status: 'OTP_SENT',
				resend_after_seconds: 30,
				otp: '123456'
			};
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockResp);

			const result = await societyApi.sendVisitorOTP('9876543210', 'vqr_token_abc');
			expect(result).toEqual(mockResp);
			expect(postSpy).toHaveBeenCalledWith('/public/visit-qr/send-otp', {
				phone: '9876543210',
				qr_token: 'vqr_token_abc'
			});
		});

		it('verifies visitor OTP and receives verification token', async () => {
			const mockResp = {
				status: 'VERIFIED',
				phone_masked: '98765****0',
				verification_token: 'vtok_signed_token'
			};
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockResp);

			const result = await societyApi.verifyVisitorOTP('9876543210', '123456', 'vqr_token_abc');
			expect(result).toEqual(mockResp);
			expect(postSpy).toHaveBeenCalledWith('/public/visit-qr/verify-otp', {
				phone: '9876543210',
				otp: '123456',
				qr_token: 'vqr_token_abc'
			});
		});

		it('submits visitor self-registration and retrieves digital pass', async () => {
			const mockPass = {
				visit_id: 'visit-123',
				pass_reference: 'VP-9B7C12',
				visitor_name: 'John Doe',
				visitor_phone_masked: '98765****0',
				society_name: 'Palm Grove',
				gate_name: 'North Gate',
				purpose: 'DELIVERY',
				status: 'REGISTERED_PENDING_GATE_VERIFICATION',
				registered_at: '2026-10-19T10:00:00Z'
			};
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockPass);

			const result = await societyApi.registerVisitorByQR({
				qr_token: 'vqr_token_abc',
				full_name: 'John Doe',
				phone: '9876543210',
				verification_token: 'vtok_signed_token',
				purpose: 'DELIVERY'
			});
			expect(result.pass_reference).toBe('VP-9B7C12');
			expect(postSpy).toHaveBeenCalledWith(
				'/public/visit-qr/register',
				expect.objectContaining({
					full_name: 'John Doe',
					qr_token: 'vqr_token_abc'
				})
			);
		});

		it('retrieves public visitor pass by reference', async () => {
			const mockPass = {
				visit_id: 'visit-123',
				pass_reference: 'VP-9B7C12',
				visitor_name: 'John Doe',
				visitor_phone_masked: '98765****0',
				society_name: 'Palm Grove',
				status: 'REGISTERED_PENDING_GATE_VERIFICATION',
				registered_at: '2026-10-19T10:00:00Z'
			};
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockPass);

			const result = await societyApi.getPublicVisitorPass('VP-9B7C12');
			expect(result).toEqual(mockPass);
			expect(getSpy).toHaveBeenCalledWith('/public/visit-qr/pass/VP-9B7C12');
		});
	});

	describe('Gatekeeper Admission and Review', () => {
		it('lists pending QR visitor registrations with masked numbers', async () => {
			const mockResp = {
				pending_registrations: [
					{
						id: 'visit-123',
						pass_reference: 'VP-9B7C12',
						visitor_name: 'John Doe',
						visitor_phone_masked: '98765****0',
						purpose: 'DELIVERY',
						status: 'REGISTERED_PENDING_GATE_VERIFICATION'
					}
				],
				total: 1
			};
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockResp);

			const result = await societyApi.listPendingQRRegistrations('soc-123', { limit: 10 });
			expect(result).toEqual(mockResp);
			expect(getSpy).toHaveBeenCalledWith('/societies/soc-123/visitors/pending-qr', {
				params: { limit: 10, offset: undefined, gate_id: undefined }
			});
		});

		it('admits visitor at gate (APPROVE decision)', async () => {
			const mockResp = {
				status: 'SUCCESS',
				message: 'Visitor admitted successfully',
				visit: { id: 'visit-123', status: 'CHECKED_IN' }
			};
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockResp as unknown as { visit: Visit; message: string });

			const result = await societyApi.admitQRVisitor('soc-123', 'visit-123', {
				decision: 'APPROVE',
				gate_id: 'gate-456'
			});
			expect(result.visit.status).toBe('CHECKED_IN');
			expect(postSpy).toHaveBeenCalledWith('/societies/soc-123/visitors/visit-123/admit-qr', {
				decision: 'APPROVE',
				gate_id: 'gate-456'
			});
		});

		it('rejects visitor at gate (REJECT decision)', async () => {
			const mockResp = {
				status: 'SUCCESS',
				message: 'Visitor admission rejected',
				visit: { id: 'visit-123', status: 'REJECTED' }
			};
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockResp as unknown as { visit: Visit; message: string });

			const result = await societyApi.admitQRVisitor('soc-123', 'visit-123', {
				decision: 'REJECT',
				gate_id: 'gate-456',
				rejection_reason: 'Photo did not match visitor'
			});
			expect(result.visit.status).toBe('REJECTED');
			expect(postSpy).toHaveBeenCalledWith('/societies/soc-123/visitors/visit-123/admit-qr', {
				decision: 'REJECT',
				gate_id: 'gate-456',
				rejection_reason: 'Photo did not match visitor'
			});
		});
	});

	describe('Society Admin Gate QR Management', () => {
		it('creates entrance QR code', async () => {
			const mockQR = {
				id: 'qr-123',
				token: 'vqr_token_abc',
				status: 'ACTIVE',
				label: 'Main Gate QR'
			};
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(mockQR);

			const result = await societyApi.createGateQR('soc-123', {
				gate_id: 'gate-456',
				label: 'Main Gate QR'
			});
			expect(result).toEqual(mockQR);
			expect(postSpy).toHaveBeenCalledWith('/societies/soc-123/gates/gate-456/qr', {
				gate_id: 'gate-456',
				label: 'Main Gate QR'
			});
		});

		it('revokes and rotates entrance QR codes', async () => {
			const mockResp = { message: 'Gate QR revoked successfully' };
			const deleteSpy = vi.spyOn(api, 'delete').mockResolvedValue(mockResp);

			const resRevoke = await societyApi.revokeGateQR('soc-123', 'qr-123');
			expect(resRevoke).toEqual(mockResp);
			expect(deleteSpy).toHaveBeenCalledWith('/societies/soc-123/visitor-qrs/qr-123');

			const rotatedQR = { id: 'qr-456', status: 'ACTIVE' };
			const postSpy = vi.spyOn(api, 'post').mockResolvedValue(rotatedQR);
			const resRotate = await societyApi.rotateGateQR('soc-123', 'qr-123');
			expect(resRotate.status).toBe('ACTIVE');
			expect(postSpy).toHaveBeenCalledWith('/societies/soc-123/visitor-qrs/qr-123/rotate', {});
		});
	});

	describe('Authenticated Visitor Self-History', () => {
		it('retrieves user visitor history and passes', async () => {
			const mockResp = {
				visits: [
					{
						id: 'visit-123',
						pass_reference: 'VP-9B7C12',
						society_name: 'Palm Grove',
						visitor_name: 'John Doe',
						status: 'CHECKED_IN'
					}
				],
				total: 1
			};
			const getSpy = vi.spyOn(api, 'get').mockResolvedValue(mockResp);

			const result = await societyApi.getMyVisitorHistory({ limit: 10, offset: 0 });
			expect(result).toEqual(mockResp);
			expect(getSpy).toHaveBeenCalledWith('/visitors/my-history', {
				params: { limit: 10, offset: 0 }
			});
		});
	});
});
