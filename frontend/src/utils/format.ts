
export const WEIGHT_DEVIATION_LIMIT = 0.03;

export function formatWeight(value?: number | null): string {
	return typeof value === 'number' && value > 0 ? `${value} kg` : '待补录';
}

/** |到厂-装车| / 装车；只有两次称重都存在时才可计算。 */
export function weightDeviationRatio(item: { loadingKg?: number | null; arrivalKg?: number | null }): number | null {
	if (typeof item.loadingKg !== 'number' || item.loadingKg <= 0 || typeof item.arrivalKg !== 'number' || item.arrivalKg <= 0) {
		return null;
	}
	return Math.abs(item.arrivalKg - item.loadingKg) / item.loadingKg;
}

export function formatWeightDeviation(item: { loadingKg?: number | null; arrivalKg?: number | null }): string {
	const ratio = weightDeviationRatio(item);
	if (ratio === null) return '待补录';
	return `${(ratio * 100).toFixed(2)}%`;
}

export function weightDeviationExceeded(item: { loadingKg?: number | null; arrivalKg?: number | null }): boolean {
	const ratio = weightDeviationRatio(item);
	return ratio !== null && ratio > WEIGHT_DEVIATION_LIMIT;
}

export function formatDate(value: string): string {
  return value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : '-';
}
export function statusTone(status: string): 'success' | 'warning' | 'danger' | 'neutral' {
	if (/approved|accepted|released|completed|signed|closed|pass|ready|online|cleared|succeeded|verified|received|active/.test(status)) return 'success';
	if (/failed|fail|rejected|critical|scrap|discard|revoked|urgent|expired|suspended/.test(status)) return 'danger';
	if (/hold|warning|review|pending|restricted|limited|quarantine|submitted|in_transit|escalated/.test(status)) return 'warning';
	return 'neutral';
}

export function daysUntil(value?: string): number | null {
	if (!value) return null;
	return Math.ceil((new Date(value).getTime() - Date.now()) / 86_400_000);
}
