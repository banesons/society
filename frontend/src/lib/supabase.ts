import { createClient } from '@supabase/supabase-js';
import { env } from '$env/dynamic/public';

const supabaseUrl = env.PUBLIC_SUPABASE_URL || 'https://glpjpqpbhdcmgmwlnkay.supabase.co';
const supabaseAnonKey = env.PUBLIC_SUPABASE_ANON_KEY || '';

// Browser-safe Supabase client initialized only with anon key
export const supabase = createClient(supabaseUrl, supabaseAnonKey, {
	auth: {
		persistSession: true,
		autoRefreshToken: true
	}
});
