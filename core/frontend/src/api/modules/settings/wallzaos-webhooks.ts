import { instance } from '@/api'
import i18n from '@/i18n'

const { t } = i18n.global

export type WallzaOSWebhookListParams = {
	page: number
	page_size: number
	status: string
	keyword: string
}

export const getWallzaOSWebhookList = (params: WallzaOSWebhookListParams) => {
	return instance.get('/mail_services/list', { params })
}

export const getWallzaOSWebhookDetail = (params: { event_id: string }) => {
	return instance.get('/mail_services/detail', { params })
}

export const retryWallzaOSWebhook = (params: { event_id: string }) => {
	return instance.post('/mail_services/update', params, {
		fetchOptions: {
			loading: 'Queueing webhook retry...',
			successMessage: true,
		},
	})
}
