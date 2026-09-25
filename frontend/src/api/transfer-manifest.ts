
import { request } from './client';
import type { DomainRecord } from '../types/domain';

export interface ManifestWeighingInput {
  expectedVersion: number;
  loadingKg?: number;
  vehiclePlate?: string;
  escortName?: string;
  arrivalKg?: number;
  reason: string;
}

export interface ManifestTransitionInput {
  status: string;
  expectedVersion: number;
  reason: string;
  loadingKg?: number;
  vehiclePlate?: string;
  escortName?: string;
  weightDeviationReason?: string;
}

export async function listTransferManifest(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/manifests?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createTransferManifest(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/manifests', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionTransferManifest(id: number, input: ManifestTransitionInput) {
  return request<DomainRecord>(`/manifests/${id}/transition`, { method: 'POST', body: JSON.stringify(input) });
}
export async function registerManifestWeighing(id: number, input: ManifestWeighingInput) {
  return request<DomainRecord>(`/manifests/${id}/weighing`, { method: 'POST', body: JSON.stringify(input) });
}
