import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ListResult } from './client.service';

export type MetodoPago = 'efectivo' | 'transferencia' | 'cheque' | 'tarjeta' | 'qr';

export interface Pago {
  id: string;
  cliente_id: string;
  prestamo_id: string;
  cuota_id?: string;
  fecha_pago: string;
  monto_pagado: number;
  capital_pagado: number;
  interes_pagado: number;
  mora_pagada: number;
  tipo: 'total' | 'parcial';
  metodo_pago: MetodoPago;
  usuario_id: string;
  numero_recibo?: string;
  observaciones?: string;
  anulado: boolean;
  anulado_at?: string;
  anulado_por?: string;
  motivo_anulacion?: string;
  created_at: string;
}

export interface CreatePagoInput {
  cuota_id: string;
  monto_pagado: number;
  metodo_pago: MetodoPago;
  usuario_id: string;
  observaciones?: string;
}

@Injectable({ providedIn: 'root' })
export class PaymentService {
  private http = inject(HttpClient);
  private base = '/api/payments';

  list(opts: { page?: number; limit?: number; prestamo_id?: string; cliente_id?: string } = {}): Observable<ListResult<Pago>> {
    const params: Record<string, string> = {};
    if (opts.page != null) params['page'] = String(opts.page);
    if (opts.limit != null) params['limit'] = String(opts.limit);
    if (opts.prestamo_id) params['prestamo_id'] = opts.prestamo_id;
    if (opts.cliente_id) params['cliente_id'] = opts.cliente_id;
    return this.http.get<ListResult<Pago>>(this.base, { params });
  }

  /**
   * Registra un pago. `idempotencyKey` debe ser estable por intento de pago
   * (no por reintento): así un reenvío del mismo pago — por timeout o doble
   * click — no genera un doble cobro; el backend reproduce la respuesta original.
   */
  create(data: CreatePagoInput, idempotencyKey?: string): Observable<{ pago: Pago; cuota: any; prestamo: any }> {
    const headers = idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : undefined;
    return this.http.post<{ pago: Pago; cuota: any; prestamo: any }>(this.base, data, { headers });
  }

  /**
   * Anula un pago (reversión compensatoria). Reservado a supervisor/admin en el
   * backend; el `motivo` queda en la auditoría. Devuelve el pago anulado y el
   * estado al que converge la cuota/préstamo.
   */
  anular(id: string, motivo: string): Observable<{ pago: Pago; cuota: any; prestamo: any }> {
    return this.http.post<{ pago: Pago; cuota: any; prestamo: any }>(`${this.base}/${id}/void`, { motivo });
  }
}
