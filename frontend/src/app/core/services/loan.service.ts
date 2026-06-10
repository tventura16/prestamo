import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ListResult } from './client.service';

export type EstadoPrestamo = 'pendiente' | 'aprobado' | 'rechazado' | 'activo' | 'finalizado' | 'mora';
export type Frecuencia = 'diaria' | 'semanal' | 'quincenal' | 'mensual';
export type SubtipoGarantia = 'vehiculo' | 'inmueble' | 'garante' | 'mueble';

export interface GarantiaImagen {
  id: string;
  garantia_id: string;
  nombre_archivo: string;
  mime: string;
  tamanio_bytes: number;
  descripcion?: string;
  subido_por?: string;
  created_at: string;
}

export interface Garantia {
  id: string;
  prestamo_id: string;
  subtipo: SubtipoGarantia;
  descripcion?: string;
  valor_estimado?: number;
  moneda: string;
  cliente_garante_id?: string;
  datos: Record<string, any>;
  created_at: string;
  updated_at: string;
  imagenes?: GarantiaImagen[];
}

export interface CreateGarantiaInput {
  subtipo: SubtipoGarantia;
  descripcion?: string;
  valor_estimado?: number;
  moneda?: string;
  cliente_garante_id?: string;
  datos: Record<string, any>;
}

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

  // ─── Garantías (entidad con datos por subtipo) ───

  listGarantias(prestamoId: string): Observable<{ total: number; items: Garantia[] }> {
    return this.http.get<{ total: number; items: Garantia[] }>(`${this.base}/${prestamoId}/garantias`);
  }

  createGarantia(prestamoId: string, data: CreateGarantiaInput): Observable<Garantia> {
    return this.http.post<Garantia>(`${this.base}/${prestamoId}/garantias`, data);
  }

  deleteGarantia(prestamoId: string, gid: string): Observable<{ deleted: string }> {
    return this.http.delete<{ deleted: string }>(`${this.base}/${prestamoId}/garantias/${gid}`);
  }

  // ─── Imágenes de una garantía ───

  /** Sube una imagen multipart. No fijamos Content-Type (el navegador agrega
   *  el boundary); el interceptor añade el Bearer. */
  uploadImagen(prestamoId: string, gid: string, file: File, descripcion?: string): Observable<GarantiaImagen> {
    const form = new FormData();
    form.append('imagen', file);
    if (descripcion) form.append('descripcion', descripcion);
    return this.http.post<GarantiaImagen>(`${this.base}/${prestamoId}/garantias/${gid}/imagenes`, form);
  }

  downloadImagen(prestamoId: string, gid: string, iid: string): Observable<Blob> {
    return this.http.get(`${this.base}/${prestamoId}/garantias/${gid}/imagenes/${iid}/download`, { responseType: 'blob' });
  }

  deleteImagen(prestamoId: string, gid: string, iid: string): Observable<{ deleted: string }> {
    return this.http.delete<{ deleted: string }>(`${this.base}/${prestamoId}/garantias/${gid}/imagenes/${iid}`);
  }
}
