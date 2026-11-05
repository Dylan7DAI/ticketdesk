// WF§9：每个 code 对应的中文提示。后端的 message 也是中文，这里作为统一入口，
// 方便前端按 code 做差异化提示（如 409 时提示"已刷新"）。
const TEXT: Record<string, string> = {
  VALIDATION_ERROR: '提交的内容有误，请检查标红的字段',
  UNAUTHENTICATED: '身份无效，请重新选择用户',
  FORBIDDEN: '当前角色无权执行此操作',
  NOT_ASSIGNEE: '只有工单处理人或主管可以执行此操作',
  TICKET_NOT_FOUND: '工单不存在',
  ALREADY_CLAIMED: '工单已被其他客服领取',
  INVALID_TRANSITION: '当前状态下不能执行该状态流转',
  INVALID_STATE: '当前工单状态不允许此操作',
  SAME_ASSIGNEE: '改派目标与当前处理人相同',
  INVALID_ASSIGNEE: '目标用户不存在或不是客服',
  INTERNAL: '服务器内部错误，请稍后重试',
  NETWORK_ERROR: '无法连接服务器，请确认后端已启动',
}

export function errorText(code: string, fallback?: string): string {
  return TEXT[code] ?? fallback ?? '操作失败'
}

// 409 表示页面上的状态已过期：提示后需要重新拉取详情。
export function isConflict(status: number): boolean {
  return status === 409
}
