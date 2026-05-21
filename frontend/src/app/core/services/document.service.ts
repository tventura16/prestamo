import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ListResult } from './client.service';

export interface Documento {
  id: string;
  tipo: 'contrato' | 'plan_pagos' | 'recibo' | 'estado_cuenta' | 'carta_mora';
  cliente_id?: string;
  prestamo_id?: string;
  pago_id?: string;
  nombre_archivo: string;
  ruta: string;
  hash_sha256?: string;
  tamanio_kb?: number;
  estado: string;
  generado_at: string;
}

@Injectable({ providedIn: 'root' })
export class DocumentService {
  private http = inject(HttpClient);
  private base = '/api/documents';

  list(opts: { page?: number; limit?: number; tipo?: string } = {}): Observable<ListResult<Documento>> {
    const params: Record<string, string> = {};
    if (opts.page != null) params['page'] = String(opts.page);
    if (opts.limit != null) params['limit'] = String(opts.limit);
    if (opts.tipo) params['tipo'] = opts.tipo;
    return this.http.get<ListResult<Documento>>(this.base, { params });
  }

  generateContract(prestamo_id: string): Observable<Documento> {
    return this.http.post<Documento>(`${this.base}/contract`, { prestamo_id });
  }

  generateReceipt(pago_id: string): Observable<Documento> {
    return this.http.post<Documento>(`${this.base}/receipt`, { pago_id });
  }

  generateStatement(cliente_id: string): Observable<Documento> {
    return this.http.post<Documento>(`${this.base}/statement`, { cliente_id });
  }

  downloadUrl(id: string): string {
    return `${this.base}/${id}/download`;
  }
}
