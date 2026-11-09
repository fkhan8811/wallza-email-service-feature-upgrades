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
	return instance.get('/wallzaos_webhooks/list', { params })
}

export const getWallzaOSWebhookDetail = (params: { event_id: string }) => {
	return instance.get('/wallzaos_webhooks/detail', { params })
}

export const retryWallzaOSWebhook = (params: { event_id: string }) => {
	return instance.post('/wallzaos_webhooks/retry', params, {
		fetchOptions: {
			loading: t('common.actions.confirm'),
			successMessage: true,
		},
	})
}
