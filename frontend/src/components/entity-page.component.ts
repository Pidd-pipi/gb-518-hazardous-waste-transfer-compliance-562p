import { AsyncPipe, CommonModule } from '@angular/common';
import { ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatInputModule } from '@angular/material/input';
import { useAuth } from '../hooks/use-auth';
import { createPagination } from '../hooks/use-pagination';
import type { EntityStore } from '../stores/factory';
import type { DomainRecord, EntityConfig } from '../types/domain';
import { TRANSITIONS } from '../types/status';
import { formatDate, formatWeight, formatWeightDeviation, weightDeviationExceeded } from '../utils/format';
import { ConfirmDialogComponent } from './common/confirm-dialog.component';
import { LicensePanelComponent } from './common/license-panel.component';
import { MetricCardComponent } from './common/metric-card.component';
import { StatusBadgeComponent } from './common/status-badge.component';

type WeighingMode = 'loading' | 'arrival';

@Component({
  selector: 'app-entity-page',
  standalone: true,
  imports: [CommonModule, AsyncPipe, FormsModule, MatButtonModule, MatInputModule, StatusBadgeComponent, MetricCardComponent, ConfirmDialogComponent, LicensePanelComponent],
  template: `
    <main class="workspace" *ngIf="store.state$ | async as state">
      <header class="page-header">
        <div><p class="eyebrow">业务工作台</p><h1>{{ config.label }}</h1><p>{{ pageDescription() }}</p></div>
        <button *ngIf="auth.hasMinimumRole('operator')" mat-flat-button color="primary" (click)="openCreate()">新增{{ config.label }}</button>
      </header>

      <app-license-panel *ngIf="isLicensePage()" [records]="state.items" />

      <section class="metrics">
        <app-metric-card label="记录总数" [value]="state.meta.total" detail="当前查询结果" />
        <app-metric-card label="高风险" [value]="highRisk(state.items)" detail="需要优先复核" />
        <app-metric-card label="状态种类" [value]="statusCount(state.items)" detail="当前页状态覆盖" />
      </section>

      <section class="toolbar">
        <input matInput aria-label="搜索" [(ngModel)]="search" [placeholder]="'搜索' + config.label + '编码或名称'" (keyup.enter)="query()" />
        <button mat-flat-button color="primary" (click)="query()">查询</button>
        <button mat-button (click)="reset()">重置</button>
      </section>
      <div *ngIf="state.error" class="alert" role="alert">{{ state.error }}</div>

      <section class="table-shell">
        <table>
          <thead><tr>
            <th>编码</th><th>名称</th><th>状态</th><th>业务凭证</th><th>风险</th><th>责任人</th>
            <th *ngIf="isManifestPage()">称重（计划 / 装车 / 到厂 / 偏差）</th>
            <th>指标</th><th>更新时间</th><th>操作</th>
          </tr></thead>
          <tbody>
            <tr *ngFor="let item of state.items; trackBy: trackById">
              <td><strong>{{ item.code }}</strong></td>
              <td>{{ item.name }}<small>{{ item.facility }}</small></td>
              <td><app-status-badge [status]="item.status" /></td>
              <td><span class="domain-detail">{{ domainDetail(item) }}</span><small>{{ item.evidence }}</small></td>
              <td><span [class]="'risk risk--' + item.riskLevel">{{ item.riskLevel }}</span></td>
              <td>{{ item.owner }}</td>
              <td *ngIf="isManifestPage()" class="weighing-cell">
                <div class="weighing-row">
                  <span class="weighing-label">计划</span><strong>{{ formatWeight(item.quantityKg) }}</strong>
                  <span class="weighing-label">装车</span><strong [class.awaiting]="!hasLoading(item)">{{ formatWeight(item.loadingKg) }}</strong>
                  <span class="weighing-label">到厂</span><strong [class.awaiting]="!hasArrival(item)">{{ formatWeight(item.arrivalKg) }}</strong>
                </div>
                <div class="weighing-row">
                  <span class="weighing-label">偏差</span>
                  <strong [class.deviation]="deviationExceeded(item)">{{ formatWeightDeviation(item) }}</strong>
                  <small *ngIf="item.vehiclePlate" class="weighing-meta">{{ item.vehiclePlate }} · {{ item.escortName }}</small>
                </div>
                <small *ngIf="item.weightDeviationReason" class="weighing-reason">偏差原因：{{ item.weightDeviationReason }}</small>
                <small *ngIf="weighingBackfillHint(item)" class="weighing-hint">{{ weighingBackfillHint(item) }}</small>
              </td>
              <td>{{ item.metricValue }} {{ item.metricUnit }}</td>
              <td>{{ formatDate(item.updatedAt) }}</td>
              <td class="actions">
                <ng-container *ngIf="canTransition()">
                  <button *ngIf="isManifestPage() && canRegisterWeighing(item)" class="table-action" (click)="openWeighing(item, weighingMode(item))">
                    {{ weighingActionLabel(item) }}
                  </button>
                  <button *ngFor="let target of transitions(item)" class="table-action" (click)="openTransition(item, target)">{{ transitionLabel(target) }}</button>
                </ng-container>
                <span *ngIf="!canTransition() || (transitions(item).length === 0 && !(isManifestPage() && canRegisterWeighing(item)))" class="muted">{{ auth.hasMinimumRole('operator') ? '流程结束' : '只读' }}</span>
              </td>
            </tr>
            <tr *ngIf="!state.items.length && !state.loading"><td [attr.colspan]="isManifestPage() ? 10 : 9" class="empty">暂无记录</td></tr>
          </tbody>
        </table>
        <div *ngIf="state.loading" class="loading">正在同步业务数据…</div>
      </section>

      <footer class="pager" *ngIf="state.meta.total > state.meta.pageSize">
        <span>第 {{ state.meta.page }} / {{ pagination.pages() }} 页</span>
        <button mat-button [disabled]="pagination.page() <= 1" (click)="previousPage()">上一页</button>
        <button mat-button [disabled]="pagination.page() >= pagination.pages()" (click)="nextPage()">下一页</button>
      </footer>

      <app-confirm-dialog [open]="showCreate" [title]="'新增' + config.label" (cancel)="closeCreate()" (confirm)="createDemo()">
        <p>确认创建一条包含责任人、业务关联、风险和证据信息的记录。</p>
      </app-confirm-dialog>

      <app-confirm-dialog *ngIf="isManifestPage()" [open]="!!weighingPending" [title]="weighingTitle()" (cancel)="closeWeighing()" (confirm)="confirmWeighing()">
        <ng-container *ngIf="weighingPending?.mode === 'loading'">
          <p>提交联单或发运前必须登记实际装车重量、车牌号和押运员，登记后不可修改。</p>
          <label class="form-grid">实际装车重量 (kg)
            <input type="number" min="0.01" step="0.01" [(ngModel)]="weighingForm.loadingKg" placeholder="例如 680.5" />
          </label>
          <label class="form-grid">车牌号
            <input [(ngModel)]="weighingForm.vehiclePlate" placeholder="例如 沪A·W2086" />
          </label>
          <label class="form-grid">押运员
            <input [(ngModel)]="weighingForm.escortName" placeholder="例如 王押运" />
          </label>
          <label class="form-grid">登记说明
            <input [(ngModel)]="weighingForm.reason" placeholder="称重地点与凭证编号" />
          </label>
        </ng-container>
        <ng-container *ngIf="weighingPending?.mode === 'arrival'">
          <p>发运后登记到厂过磅重量，登记后不可修改。与装车重量相差超过 3% 时，签收还需填写偏差原因。</p>
          <label class="form-grid">到厂重量 (kg)
            <input type="number" min="0.01" step="0.01" [(ngModel)]="weighingForm.arrivalKg" placeholder="例如 672.0" />
          </label>
          <label class="form-grid">登记说明
            <input [(ngModel)]="weighingForm.reason" placeholder="到厂过磅单编号" />
          </label>
        </ng-container>
      </app-confirm-dialog>

      <app-confirm-dialog [open]="!!pending" [title]="transitionDialogTitle()" (cancel)="closeTransition()" (confirm)="confirmTransition()">
        <p *ngIf="!isManifestPage()">状态迁移会校验关联资质，并与请求 ID 审计记录在同一事务中保存。</p>
        <ng-container *ngIf="isManifestPage() && pending as p">
          <p>状态迁移会校验关联资质，并与请求 ID 审计记录在同一事务中保存。</p>
          <strong>{{ p.item.status }} → {{ p.status }}</strong>
          <ng-container *ngIf="p.status === 'submitted' && !hasLoading(p.item)">
            <p class="form-hint">提交联单必须填写实际装车重量、车牌号和押运员；未填写或重量不大于零无法提交。</p>
            <label class="form-grid">实际装车重量 (kg)
              <input type="number" min="0.01" step="0.01" [(ngModel)]="transitionForm.loadingKg" placeholder="例如 680.5" />
            </label>
            <label class="form-grid">车牌号
              <input [(ngModel)]="transitionForm.vehiclePlate" placeholder="例如 沪A·W2086" />
            </label>
            <label class="form-grid">押运员
              <input [(ngModel)]="transitionForm.escortName" placeholder="例如 王押运" />
            </label>
          </ng-container>
          <p *ngIf="p.status === 'submitted' && hasLoading(p.item)" class="form-hint">装车称重已登记：{{ p.item.loadingKg }} kg · {{ p.item.vehiclePlate }} · {{ p.item.escortName }}</p>
          <p *ngIf="p.status === 'in_transit' && !hasLoading(p.item)" class="form-hint form-hint--danger">尚未登记装车称重，请先通过“补录装车称重”补齐后再发运。</p>
          <p *ngIf="p.status === 'in_transit' && hasLoading(p.item)" class="form-hint">发运车辆 {{ p.item.vehiclePlate }}，押运员 {{ p.item.escortName }}。</p>
          <ng-container *ngIf="p.status === 'received'">
            <p *ngIf="!hasArrival(p.item)" class="form-hint form-hint--danger">尚未登记到厂重量，联单将保持在途状态；请先通过“登记到厂重量”称重。</p>
            <ng-container *ngIf="hasArrival(p.item)">
              <p class="form-hint">计划 {{ p.item.quantityKg }} kg · 装车 {{ p.item.loadingKg }} kg · 到厂 {{ p.item.arrivalKg }} kg · 偏差 {{ formatWeightDeviation(p.item) }}</p>
              <label *ngIf="deviationExceeded(p.item)" class="form-grid">偏差原因（超过 3%，必填）
                <textarea rows="3" [(ngModel)]="transitionForm.weightDeviationReason" placeholder="说明途耗、磅差或其他原因及凭证"></textarea>
              </label>
              <p *ngIf="deviationExceeded(p.item) && p.item.weightDeviationReason" class="form-hint">已登记偏差原因：{{ p.item.weightDeviationReason }}</p>
            </ng-container>
          </ng-container>
          <label class="form-grid">流转说明
            <input [(ngModel)]="transitionForm.reason" placeholder="本次状态流转说明（至少 3 个字）" />
          </label>
        </ng-container>
        <strong *ngIf="!isManifestPage()">{{ pending?.item?.status }} → {{ pending?.status }}</strong>
      </app-confirm-dialog>
    </main>
  `
})
export class EntityPageComponent implements OnInit {
  @Input({ required: true }) config!: EntityConfig;
  @Input({ required: true }) store!: EntityStore;
  readonly auth = useAuth();
  readonly formatDate = formatDate;
  readonly formatWeight = formatWeight;
  readonly formatWeightDeviation = formatWeightDeviation;
  readonly pagination = createPagination(() => this.store?.snapshot.meta.total ?? 0, 10);
  search = '';
  showCreate = false;
  pending: { item: DomainRecord; status: string } | null = null;
  weighingPending: { item: DomainRecord; mode: WeighingMode } | null = null;
  transitionForm = { loadingKg: null as number | null, vehiclePlate: '', escortName: '', weightDeviationReason: '', reason: '' };
  weighingForm = { loadingKg: null as number | null, vehiclePlate: '', escortName: '', arrivalKg: null as number | null, reason: '' };

  constructor(private readonly changeDetector: ChangeDetectorRef) {}

  async ngOnInit(): Promise<void> { await this.load(); }
  trackById(_index: number, item: DomainRecord): number { return item.id; }
  highRisk(items: DomainRecord[]): number { return items.filter((item) => ['high', 'critical'].includes(item.riskLevel)).length; }
  statusCount(items: DomainRecord[]): number { return new Set(items.map((item) => item.status)).size; }
  isLicensePage(): boolean { return this.config.key === 'wasteGenerator' || this.config.key === 'carrierProfile'; }
  isManifestPage(): boolean { return this.config.key === 'transferManifest'; }
  canTransition(): boolean { return this.auth.hasMinimumRole(this.config.transitionRole); }
  transitions(item: DomainRecord): readonly string[] { return TRANSITIONS[this.config.key]?.[item.status] ?? []; }
  hasLoading(item: DomainRecord): boolean {
    return typeof item.loadingKg === 'number' && item.loadingKg > 0 && !!item.vehiclePlate && !!item.escortName;
  }
  hasArrival(item: DomainRecord): boolean { return typeof item.arrivalKg === 'number' && item.arrivalKg > 0; }
  deviationExceeded(item: DomainRecord): boolean { return weightDeviationExceeded(item); }

  pageDescription(): string {
    const descriptions: Record<string, string> = {
      wasteGenerator: '核对产废许可有效期、废物类别与证据文件。',
      carrierProfile: '复核承运许可证、有效车辆和资质证据。',
      transferManifest: '登记装车与到厂称重，跟踪联单提交、发运和签收全流程。',
      complianceCheck: '基于联单证据形成不可回退的核验决定。'
    };
    return descriptions[this.config.key] || `管理${this.config.label}状态和证据。`;
  }

  domainDetail(item: DomainRecord): string {
    if (this.config.key === 'wasteGenerator') return `${item.permitNumber || '-'} · ${item.wasteCategories || '-'}`;
    if (this.config.key === 'carrierProfile') return `${item.licenseNumber || '-'} · ${item.vehicleCount || 0} 辆`;
    if (this.config.key === 'transferManifest') return `${item.generatorCode} → ${item.carrierCode} · ${item.wasteCode}`;
    return `${item.manifestCode || '-'} · ${item.decisionBasis || '待决定'}`;
  }

  weighingBackfillHint(item: DomainRecord): string {
    if (item.status === 'draft' && !this.hasLoading(item)) return '待补录：提交前补齐装车称重、车牌和押运员';
    if (item.status === 'submitted' && !this.hasLoading(item)) return '待补录：发运前补齐装车称重、车牌和押运员';
    if (item.status === 'in_transit' && !this.hasArrival(item)) return '待补录：发运后登记到厂重量';
    if (item.status === 'in_transit' && this.hasArrival(item) && this.deviationExceeded(item) && !item.weightDeviationReason) return '到厂偏差超过 3%，签收时必须填写偏差原因';
    return '';
  }

  canRegisterWeighing(item: DomainRecord): boolean {
    if (item.status === 'draft' || item.status === 'submitted') return !this.hasLoading(item);
    if (item.status === 'in_transit') return !this.hasArrival(item);
    return false;
  }

  weighingMode(item: DomainRecord): WeighingMode {
    return item.status === 'in_transit' ? 'arrival' : 'loading';
  }

  weighingActionLabel(item: DomainRecord): string {
    return this.weighingMode(item) === 'arrival' ? '登记到厂重量' : '补录装车称重';
  }

  weighingTitle(): string {
    return this.weighingPending?.mode === 'arrival' ? '到厂称重登记' : '装车称重登记';
  }

  transitionLabel(status: string): string {
    const labels: Record<string, string> = { submitted: '提交', in_transit: '发运', received: '签收', rejected: '驳回', verified: '核准', restricted: '限制', expired: '到期', active: '恢复', suspended: '停用', pass: '通过', fail: '不通过', escalated: '升级复核' };
    return labels[status] || status;
  }

  transitionDialogTitle(): string {
    return `确认${this.transitionLabel(this.pending?.status ?? '')}联单`;
  }

  async query(): Promise<void> { this.pagination.reset(); await this.load(); }
  async reset(): Promise<void> { this.search = ''; this.pagination.reset(); await this.load(); }
  async previousPage(): Promise<void> { this.pagination.previous(); await this.load(); }
  async nextPage(): Promise<void> { this.pagination.next(); await this.load(); }
  openCreate(): void { this.showCreate = true; this.changeDetector.detectChanges(); }
  closeCreate(): void { this.showCreate = false; this.changeDetector.detectChanges(); }
  openTransition(item: DomainRecord, status: string): void {
    this.pending = { item, status };
    this.transitionForm = { loadingKg: null, vehiclePlate: '', escortName: '', weightDeviationReason: '', reason: '' };
    this.changeDetector.detectChanges();
  }
  closeTransition(): void { this.pending = null; this.changeDetector.detectChanges(); }
  openWeighing(item: DomainRecord, mode: WeighingMode): void {
    this.weighingPending = { item, mode };
    this.weighingForm = { loadingKg: null, vehiclePlate: '', escortName: '', arrivalKg: null, reason: '' };
    this.changeDetector.detectChanges();
  }
  closeWeighing(): void { this.weighingPending = null; this.changeDetector.detectChanges(); }

  async createDemo(): Promise<void> {
    const now = Date.now();
    const common: Partial<DomainRecord> = {
      code: `${this.config.key.toUpperCase()}-${String(now).slice(-6)}`,
      name: `新增${this.config.label}`,
      description: '通过合规工作台创建的业务记录', facility: '东区危废暂存区', owner: '现场操作员',
      category: '危废转运', riskLevel: 'medium', metricValue: 25, metricUnit: 'score',
      effectiveAt: new Date().toISOString(), evidence: `minio://evidence/${this.config.path}/${now}.pdf`, relatedCode: ''
    };
    const expiresAt = new Date(now + 365 * 86_400_000).toISOString();
    const specific: Partial<DomainRecord> = this.config.key === 'wasteGenerator'
      ? { permitNumber: `PERMIT-${String(now).slice(-8)}`, permitExpiresAt: expiresAt, wasteCategories: 'HW08 废矿物油' }
      : this.config.key === 'carrierProfile'
        ? { licenseNumber: `CARRIER-${String(now).slice(-8)}`, licenseExpiresAt: expiresAt, vehicleCount: 6 }
        : this.config.key === 'transferManifest'
          ? { generatorCode: 'WG-001', carrierCode: 'CP-002', wasteCode: 'HW08-900-249-08', quantityKg: 640, destination: '合规处置中心 A' }
          : { manifestCode: 'TM-002', checklist: '产废许可、承运资质、联单数量、处置去向', decisionBasis: '' };
    try {
      await this.store.createRecord(this.config.path, { ...common, ...specific });
      this.showCreate = false;
    } catch { /* Store exposes the request error in its observable state. */ }
    finally { this.changeDetector.detectChanges(); }
  }

  async confirmTransition(): Promise<void> {
    if (!this.pending) return;
    const { item, status } = this.pending;
    if (this.isManifestPage() && this.transitionForm.reason.trim().length < 3) {
      this.patchLocalError('请填写本次流转说明（至少 3 个字）。');
      return;
    }
    if (this.isManifestPage()) {
      if (status === 'submitted' && !this.hasLoading(item)) {
        if (!(Number(this.transitionForm.loadingKg) > 0) || !this.transitionForm.vehiclePlate.trim() || !this.transitionForm.escortName.trim()) {
          this.patchLocalError('提交联单必须填写大于零的实际装车重量、车牌号和押运员。');
          return;
        }
      }
      if (status === 'in_transit' && !this.hasLoading(item)) {
        this.patchLocalError('请先补录装车称重、车牌号和押运员，再执行发运。');
        return;
      }
      if (status === 'received') {
        if (!this.hasArrival(item)) {
          this.patchLocalError('请先登记到厂重量；未称重的联单保持在途状态。');
          return;
        }
        if (this.deviationExceeded(item) && !item.weightDeviationReason && !this.transitionForm.weightDeviationReason.trim()) {
          this.patchLocalError('到厂重量与装车重量偏差超过 3%，必须填写偏差原因才能签收。');
          return;
        }
      }
    }
    const extra: Record<string, unknown> = {};
    if (this.isManifestPage() && this.transitionForm.reason.trim()) extra.reason = this.transitionForm.reason.trim();
    if (this.isManifestPage() && status === 'submitted' && !this.hasLoading(item)) {
      extra.loadingKg = Number(this.transitionForm.loadingKg);
      extra.vehiclePlate = this.transitionForm.vehiclePlate.trim();
      extra.escortName = this.transitionForm.escortName.trim();
    }
    if (this.isManifestPage() && status === 'received' && this.transitionForm.weightDeviationReason.trim()) {
      extra.weightDeviationReason = this.transitionForm.weightDeviationReason.trim();
    }
    try {
      await this.store.transition(this.config.path, item, status, extra);
      this.pending = null;
    } catch { /* Store exposes the request error in its observable state. */ }
    finally { this.changeDetector.detectChanges(); }
  }

  async confirmWeighing(): Promise<void> {
    if (!this.weighingPending) return;
    const { item, mode } = this.weighingPending;
    if (!this.weighingForm.reason.trim() || this.weighingForm.reason.trim().length < 3) {
      this.patchLocalError('请填写称重登记说明（至少 3 个字）。');
      return;
    }
    const payload: Record<string, unknown> = { reason: this.weighingForm.reason.trim() };
    if (mode === 'loading') {
      if (!(Number(this.weighingForm.loadingKg) > 0) || !this.weighingForm.vehiclePlate.trim() || !this.weighingForm.escortName.trim()) {
        this.patchLocalError('装车称重必须填写大于零的重量、车牌号和押运员。');
        return;
      }
      payload.loadingKg = Number(this.weighingForm.loadingKg);
      payload.vehiclePlate = this.weighingForm.vehiclePlate.trim();
      payload.escortName = this.weighingForm.escortName.trim();
    } else {
      if (!(Number(this.weighingForm.arrivalKg) > 0)) {
        this.patchLocalError('到厂重量必须大于零。');
        return;
      }
      payload.arrivalKg = Number(this.weighingForm.arrivalKg);
    }
    try {
      await this.store.registerWeighing(this.config.path, item, payload);
      this.weighingPending = null;
    } catch { /* Store exposes the request error in its observable state. */ }
    finally { this.changeDetector.detectChanges(); }
  }

  private patchLocalError(message: string): void {
    this.store.patchError(message);
    this.changeDetector.detectChanges();
  }

  private async load(): Promise<void> {
    await this.store.load(this.config.path, this.search, this.pagination.page(), this.pagination.pageSize());
    this.changeDetector.detectChanges();
  }
}
