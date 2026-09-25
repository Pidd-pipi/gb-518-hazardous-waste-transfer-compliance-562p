
import { CommonModule } from '@angular/common';
import { Component, EventEmitter, Input, OnChanges, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import type { DomainRecord } from '../../types/domain';

export type WeighingDialogMode = 'loading' | 'arrival' | 'signoff';

@Component({
  selector: 'app-manifest-weighing-dialog',
  standalone: true,
  imports: [CommonModule, FormsModule, MatButtonModule],
  template: `
    <div *ngIf="open" class="modal-backdrop" (click)="cancel.emit()">
      <section class="modal modal--weighing" role="dialog" aria-modal="true" aria-labelledby="weighing-title" (click)="$event.stopPropagation()">
        <h2 id="weighing-title">{{ title() }}</h2>

        <ng-container *ngIf="item as manifest">
          <p class="weighing-summary">
            <span>联单 <strong>{{ manifest.code }}</strong></span>
            <span>计划重量 <strong>{{ manifest.quantityKg | number: '0.###' }} kg</strong></span>
          </p>

          <ng-container *ngIf="mode === 'loading'">
            <label>实际装车重量（kg）<input type="number" min="0" step="0.01" [(ngModel)]="loadWeightKg" [readonly]="!!manifest.loadWeightKg" placeholder="例如 680.5" /></label>
            <label>车牌号<input type="text" [(ngModel)]="vehiclePlate" [readonly]="!!manifest.loadWeightKg" maxlength="32" placeholder="例如 鲁B·W8052" /></label>
            <label>押运员<input type="text" [(ngModel)]="escortName" [readonly]="!!manifest.loadWeightKg" maxlength="80" placeholder="押运员姓名" /></label>
            <p class="weighing-hint">提交联单前必须登记实际装车重量、车牌号和押运员，重量须大于 0。</p>
          </ng-container>

          <ng-container *ngIf="mode === 'arrival'">
            <p class="weighing-summary">
              <span>装车重量 <strong>{{ manifest.loadWeightKg | number: '0.###' }} kg</strong></span>
              <span *ngIf="manifest.vehiclePlate">{{ manifest.vehiclePlate }} · {{ manifest.escortName }}</span>
            </p>
            <label>到厂重量（kg）<input type="number" min="0" step="0.01" [(ngModel)]="arrivalWeightKg" [readonly]="!!manifest.arrivalWeightKg" placeholder="例如 700.9" /></label>
            <p class="weighing-result" [class.weighing-result--danger]="deviationExceeds()">
              装车 / 到厂偏差：<strong>{{ deviationText() }}</strong>
            </p>
            <label *ngIf="deviationExceeds()">偏差原因（偏差超过 3%，必填）
              <textarea rows="3" [(ngModel)]="deviationReason" maxlength="500" placeholder="请说明到厂重量与装车重量差异超过 3% 的原因"></textarea>
            </label>
            <p class="weighing-hint">登记到厂重量不会改变联单状态，联单继续保留在途；补齐后方可签收。</p>
          </ng-container>

          <ng-container *ngIf="mode === 'signoff'">
            <p class="weighing-summary">
              <span>装车重量 <strong>{{ weightText(manifest.loadWeightKg) }}</strong></span>
              <span>到厂重量 <strong>{{ weightText(manifest.arrivalWeightKg) }}</strong></span>
            </p>
            <p *ngIf="manifest.loadWeightKg == null" class="weighing-error">装车称重尚未补录，请先取消签收并使用“补录装车”登记实际装车重量、车牌号和押运员。</p>
            <p *ngIf="manifest.arrivalWeightKg == null" class="weighing-error">到厂重量尚未登记，请先使用“登记到厂”补录，联单暂不能签收。</p>
            <ng-container *ngIf="manifest.arrivalWeightKg != null">
              <p class="weighing-result" [class.weighing-result--danger]="deviationExceeds()">
                装车 / 到厂偏差：<strong>{{ deviationText() }}</strong>
              </p>
              <label *ngIf="deviationExceeds() && !manifest.weightDeviationReason">偏差原因（偏差超过 3%，必填）
                <textarea rows="3" [(ngModel)]="deviationReason" maxlength="500" placeholder="请说明偏差原因后再签收"></textarea>
              </label>
              <p *ngIf="deviationExceeds() && manifest.weightDeviationReason" class="weighing-hint">已登记偏差原因：{{ manifest.weightDeviationReason }}</p>
            </ng-container>
          </ng-container>
        </ng-container>

        <footer>
          <button mat-button (click)="cancel.emit()">取消</button>
          <button mat-flat-button color="primary" [disabled]="!valid()" (click)="emitConfirm()">{{ confirmLabel() }}</button>
        </footer>
      </section>
    </div>
  `
})
export class ManifestWeighingDialogComponent implements OnChanges {
  @Input() open = false;
  @Input() mode: WeighingDialogMode = 'loading';
  @Input() item: DomainRecord | null = null;
  @Output() confirm = new EventEmitter<Record<string, unknown>>();
  @Output() cancel = new EventEmitter<void>();

  loadWeightKg: number | null = null;
  arrivalWeightKg: number | null = null;
  vehiclePlate = '';
  escortName = '';
  deviationReason = '';

  ngOnChanges(): void {
    if (!this.open || !this.item) return;
    this.loadWeightKg = this.item.loadWeightKg ?? null;
    this.arrivalWeightKg = this.item.arrivalWeightKg ?? null;
    this.vehiclePlate = this.item.vehiclePlate || '';
    this.escortName = this.item.escortName || '';
    this.deviationReason = '';
  }

  title(): string {
    if (this.mode === 'loading') return '补录装车称重';
    if (this.mode === 'arrival') return this.item?.arrivalWeightKg == null ? '登记到厂重量' : '补录偏差原因';
    return '联单签收确认';
  }

  confirmLabel(): string {
    return this.mode === 'signoff' ? '确认签收' : '保存称重登记';
  }

  weightText(value?: number | null): string {
    return value == null ? '待补录' : `${this.round(value)} kg`;
  }

  deviationPct(): number | null {
    const load = this.item?.loadWeightKg;
    const arrival = this.arrivalWeightKg ?? this.item?.arrivalWeightKg ?? null;
    if (load == null || arrival == null || load <= 0) return null;
    return ((arrival - load) / load) * 100;
  }

  deviationText(): string {
    const deviation = this.deviationPct();
    return deviation == null ? '待补录' : `${deviation > 0 ? '+' : ''}${this.round(deviation)}%`;
  }

  deviationExceeds(): boolean {
    const deviation = this.deviationPct();
    return deviation != null && Math.abs(deviation) > 3;
  }

  valid(): boolean {
    if (!this.item) return false;
    if (this.mode === 'loading') {
      return this.loadWeightKg != null && this.loadWeightKg > 0
        && this.vehiclePlate.trim() !== '' && this.escortName.trim() !== '';
    }
    if (this.mode === 'arrival') {
      if (this.arrivalWeightKg == null || this.arrivalWeightKg <= 0) return false;
      return !this.deviationExceeds() || this.deviationReason.trim() !== '';
    }
    if (this.item.arrivalWeightKg == null) return false;
    return !this.deviationExceeds() || !!this.item.weightDeviationReason || this.deviationReason.trim() !== '';
  }

  emitConfirm(): void {
    if (!this.valid()) return;
    if (this.mode === 'loading') {
      this.confirm.emit({
        loadWeightKg: this.loadWeightKg,
        vehiclePlate: this.vehiclePlate.trim(),
        escortName: this.escortName.trim(),
      });
      return;
    }
    if (this.mode === 'arrival') {
      const payload: Record<string, unknown> = { arrivalWeightKg: this.arrivalWeightKg };
      if (this.deviationReason.trim()) payload.weightDeviationReason = this.deviationReason.trim();
      this.confirm.emit(payload);
      return;
    }
    const payload: Record<string, unknown> = {};
    if (this.deviationReason.trim()) payload.weightDeviationReason = this.deviationReason.trim();
    this.confirm.emit(payload);
  }

  private round(value: number): number {
    return Math.round(value * 1000) / 1000;
  }
}
