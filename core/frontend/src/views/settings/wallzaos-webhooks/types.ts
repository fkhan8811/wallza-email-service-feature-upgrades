export type WallzaOSWebhookDelivery = {
	event_id: string
	event_type: string
	status: 'pending' | 'processing' | 'delivered' | 'failed' | string
	attempts: number
	next_attempt_at: number
	created_at: number
	delivered_at: number
	last_error: string
}
