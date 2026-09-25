
import { request } from './client';
import type { DomainRecord, ManifestTransitionInput, ManifestWeighingInput } from '../types/domain';

export async function listTransferManifest(page = 1, pageSize = 20, search = '') {
  return request<DomainRecord[]>(`/manifests?page=${page}&pageSize=${pageSize}&search=${encodeURIComponent(search)}`);
}
export async function createTransferManifest(input: Partial<DomainRecord>) {
  return request<DomainRecord>('/manifests', { method: 'POST', body: JSON.stringify(input) });
}
export async function transitionTransferManifest(id: number, input: ManifestTransitionInput) {
  return request<DomainRecord>(`/manifests/${id}/transition`, {
    method: 'POST', body: JSON.stringify(input),
  });
}
export async function registerTransferManifestWeighing(id: number, input: ManifestWeighingInput) {
  return request<DomainRecord>(`/manifests/${id}/weighing`, {
    method: 'POST', body: JSON.stringify(input),
  });
}
