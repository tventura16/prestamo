import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ListResult } from './client.service';

export type EstadoPrestamo = 'pendiente' | 'aprobado' | 'rechazado' | 'activo' | 'finalizado' | 'mora';
export type Frecuencia = 'diaria' | 'semanal' | 'quincenal' | 'mensual';

export interface Prestamo {
  id: string;
  cliente_id: string;
  monto_solicitado: number;
  monto_aprobado?: number;
  tasa_interes: number;
  tipo_interes: 'fijo' | 'variable';
  fecha_solicitud: string;
  fecha_desembolso?: string;
  num_cuotas: number;
  frecuencia: Frecuencia;
  estado: EstadoPrestamo;
  aprobado_por?: string;
  observaciones?: string;
  created_at: string;
  updated_at: string;
}

export interface Cuota {
  id: string;
  prestamo_id: string;
  numero: number;
  fecha_vencimiento: string;
  capital: number;
  interes: number;
  total: number;
  saldo_pendiente: number;
  mora_acumulada: number;
  estado: 'pendiente' | 'pagada' | 'parcial' | 'vencida';
  fecha_pago?: string;
}

export interface CreatePrestamoInput {
  cliente_id: string;
  monto_solicitado: number;
  tasa_interes: number;
  num_cuotas: number;
  frecuencia: Frecuencia;
  observaciones?: string;
}

export interface ApproveInput {
  monto_aprobado?: number;
  fecha_desembolso?: string;
  aprobado_por: string;
  observaciones?: string;
}

@Injectable({ providedIn: 'root' })
export class LoanService {
  private http = inject(HttpClient);
  private base = '/api/loans';

  list(opts: { page?: number; limit?: number; cliente_id?: string; estado?: EstadoPrestamo } = {}): Observable<ListResult<Prestamo>> {
    const params: Record<string, string> = {};
    if (opts.page != null) params['page'] = String(opts.page);
    if (opts.limit != null) params['limit'] = String(opts.limit);
    if (opts.cliente_id) params['cliente_id'] = opts.cliente_id;
    if (opts.estado) params['estado'] = opts.estado;
    return this.http.get<ListResult<Prestamo>>(this.base, { params });
  }

  get(id: string): Observable<Prestamo> {
    return this.http.get<Prestamo>(`${this.base}/${id}`);
  }

  schedule(id: string): Observable<{ cuotas: Cuota[] }> {
    return this.http.get<{ cuotas: Cuota[] }>(`${this.base}/${id}/schedule`);
  }

  create(data: CreatePrestamoInput): Observable<Prestamo> {
    return this.http.post<Prestamo>(this.base, data);
  }

  approve(id: string, data: ApproveInput): Observable<{ prestamo: Prestamo; cuotas: Cuota[] }> {
    return this.http.post<{ prestamo: Prestamo; cuotas: Cuota[] }>(`${this.base}/${id}/approve`, data);
  }

  reject(id: string, data: { aprobado_por: string; observaciones: string }): Observable<Prestamo> {
    return this.http.post<Prestamo>(`${this.base}/${id}/reject`, data);
  }
}
