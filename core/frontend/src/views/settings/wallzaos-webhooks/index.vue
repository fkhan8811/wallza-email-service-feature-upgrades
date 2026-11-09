<template>
	<bt-table-layout>
		<template #toolsLeft>
			<n-input
				v-model:value="tableParams.keyword"
				clearable
				style="width: 240px"
				placeholder="Search event ID or event type"
				@keyup.enter="handleRefresh"
			/>
			<n-select
				v-model:value="tableParams.status"
				clearable
				style="width: 160px"
				:options="statusOptions"
				placeholder="All statuses"
				@update:value="handleRefresh"
			/>
			<n-button type="primary" @click="handleRefresh">Search</n-button>
		</template>
		<template #toolsRight>
			<n-button @click="handleRefresh">Refresh</n-button>
		</template>
		<template #table>
			<n-data-table :loading="loading" :columns="columns" :data="tableList">
				<template #empty>
					<bt-table-help>No WallzaOS webhook deliveries found.</bt-table-help>
				</template>
			</n-data-table>
		</template>
		<template #pageRight>
			<bt-table-page
				v-model:page="tableParams.page"
				v-model:page-size="tableParams.page_size"
				:item-count="tableTotal"
				@refresh="getTableData"
			/>
		</template>
		<template #modal>
			<bt-modal v-model:show="detailVisible" :width="860" :title="detailTitle" :footer="false">
				<n-spin :show="detailLoading">
					<div class="mb-12px flex flex-wrap gap-12px text-13px">
						<span><b>Status:</b> {{ detailDelivery?.status || '—' }}</span>
						<span><b>Attempts:</b> {{ detailDelivery?.attempts ?? 0 }}</span>
						<span><b>Created:</b> {{ detailDelivery ? formatTime(detailDelivery.created_at) : '—' }}</span>
						<span v-if="detailDelivery?.delivered_at">
							<b>Delivered:</b> {{ formatTime(detailDelivery.delivered_at) }}
						</span>
					</div>
					<div v-if="detailDelivery?.last_error" class="mb-12px text-13px text-error">
						<b>Last error:</b> {{ detailDelivery.last_error }}
					</div>
					<bt-logs height="480px" :code="detailPayload" />
				</n-spin>
			</bt-modal>
		</template>
	</bt-table-layout>
</template>

<script lang="tsx" setup>
import { DataTableColumns, NButton, NFlex, NTag } from 'naive-ui'
import { confirm, formatTime } from '@/utils'
import { useTableData } from '@/hooks/useTableData'
import {
	getWallzaOSWebhookList,
	getWallzaOSWebhookDetail,
	retryWallzaOSWebhook,
	type WallzaOSWebhookListParams,
} from '@/api/modules/settings/wallzaos-webhooks'
import type { WallzaOSWebhookDelivery } from './types'

const { tableParams, tableList, loading, tableTotal, getTableData } = useTableData<
	WallzaOSWebhookDelivery,
	WallzaOSWebhookListParams
>({
	loading: true,
	immediate: true,
	params: {
		page: 1,
		page_size: 20,
		status: '',
		keyword: '',
	},
	fetchFn: getWallzaOSWebhookList,
})

const detailVisible = ref(false)
const detailLoading = ref(false)
const detailTitle = ref('Webhook delivery')
const detailPayload = ref('')
const detailDelivery = ref<WallzaOSWebhookDelivery | null>(null)

const statusOptions = [
	{ label: 'Pending', value: 'pending' },
	{ label: 'Processing', value: 'processing' },
	{ label: 'Delivered', value: 'delivered' },
	{ label: 'Failed', value: 'failed' },
]

const statusTagType = (status: string): 'success' | 'error' | 'warning' | 'info' => {
	if (status === 'delivered') return 'success'
	if (status === 'failed') return 'error'
	if (status === 'processing') return 'warning'
	return 'info'
}

const columns = ref<DataTableColumns<WallzaOSWebhookDelivery>>([
	{
		key: 'event_id',
		title: 'Delivery ID',
		width: 190,
		ellipsis: { tooltip: true },
		render: row => (
			<NButton text type="primary" onClick={() => openDetail(row)}>
				{row.event_id}
			</NButton>
		),
	},
	{
		key: 'event_type',
		title: 'Event',
		width: 190,
		ellipsis: { tooltip: true },
	},
	{
		key: 'status',
		title: 'Status',
		width: 110,
		render: row => <NTag type={statusTagType(row.status)}>{row.status}</NTag>,
	},
	{
		key: 'attempts',
		title: 'Attempts',
		width: 85,
	},
	{
		key: 'created_at',
		title: 'Created',
		width: 160,
		render: row => formatTime(row.created_at),
	},
	{
		key: 'delivered_at',
		title: 'Delivered',
		width: 160,
		render: row => (row.delivered_at > 0 ? formatTime(row.delivered_at) : '—'),
	},
	{
		key: 'last_error',
		title: 'Last error',
		ellipsis: { tooltip: true },
		render: row => row.last_error || '—',
	},
	{
		key: 'actions',
		title: 'Actions',
		align: 'right',
		width: 130,
		render: row => (
			<NFlex inline>
				<NButton text type="primary" onClick={() => openDetail(row)}>
					View
				</NButton>
				{row.status === 'failed' ? (
					<NButton text type="warning" onClick={() => handleRetry(row)}>
						Retry
					</NButton>
				) : null}
			</NFlex>
		),
	},
])

const handleRefresh = () => {
	getTableData(true)
}

const openDetail = async (row: WallzaOSWebhookDelivery) => {
	detailVisible.value = true
	detailLoading.value = true
	detailTitle.value = `Webhook delivery · ${row.event_id}`
	detailDelivery.value = row
	detailPayload.value = ''
	try {
		const result = await getWallzaOSWebhookDetail({ event_id: row.event_id })
		detailDelivery.value = result.delivery || row
		try {
			const parsed = JSON.parse(result.payload || '{}')
			detailPayload.value = JSON.stringify(parsed, null, 2)
		} catch {
			detailPayload.value = result.payload || ''
		}
	} finally {
		detailLoading.value = false
	}
}

const handleRetry = (row: WallzaOSWebhookDelivery) => {
	confirm({
		title: 'Retry failed webhook delivery?',
		content: `Retry ${row.event_type} · ${row.event_id} now?`,
		onConfirm: async () => {
			await retryWallzaOSWebhook({ event_id: row.event_id })
			getTableData()
		},
	})
}
</script>

<style lang="scss" scoped></style>
