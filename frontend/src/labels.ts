import type { Action, Priority, Status } from './api/types'

// 显示用中文名（WF§17.3）和 SLA 时长（WF§8）。

export const STATUS_LABEL: Record<Status, string> = {
  open: '待处理', in_progress: '处理中', resolved: '已解决', closed: '已关闭',
}

export const PRIORITY_LABEL: Record<Priority, string> = {
  urgent: '紧急', high: '高', normal: '普通', low: '低',
}

export const PRIORITY_SLA: Record<Priority, string> = {
  urgent: '2 小时', high: '8 小时', normal: '24 小时', low: '72 小时',
}

export const ACTION_LABEL: Record<Action, string> = {
  claim: '领取', assign: '分配', reassign: '改派',
  resolve: '解决', reopen: '重开', close: '关闭', comment: '评论',
}

export const STATUS_ORDER: Status[] = ['open', 'in_progress', 'resolved', 'closed']
export const PRIORITY_ORDER: Priority[] = ['urgent', 'high', 'normal', 'low']

export function formatTime(iso: string): string {
  return new Date(iso).toLocaleString('zh-CN', { hour12: false })
}
