import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';

export interface DashboardData {
  prestamos_activos: number;
  prestamos_en_mora: number;
  clientes_activos: number;
  cuotas_vencidas: number;
  ingresos_mes: number;
  ingresos_hoy: number;
  cartera_outstanding: number;
}

export interface CuotaVencida {
  cuota_id: string;
  prestamo_id: string;
  cliente_id: string;
  numero: number;
  fecha_vencimiento: string;
  dias_vencidos: number;
  total: number;
  saldo_pendiente: number;
  mora_acumulada: number;
  estado: string;
}

export interface ReporteDiario {
  fecha: string;
  ingresos: number;
  pagos_recibidos: number;
  mora_cobrada: number;
  prestamos_nuevos: number;
  clientes_nuevos: number;
}

export interface ReporteMensual {
  anio: number;
  mes: number;
  ingresos: number;
  intereses_pagados: number;
  mora_cobrada: number;
  prestamos_nuevos: number;
  clientes_nuevos: number;
  pagos_recibidos: number;
}

export interface PrestamoResumen {
  id: string;
  monto_aprobado?: number;
  estado: string;
  num_cuotas: number;
  cuotas_pagadas: number;
  cuotas_vencidas: number;
  saldo_pendiente: number;
  total_pagado: number;
  fecha_solicitud: string;
}

export interface ClienteInfoReporte {
  nombres: string;
  apellidos: string;
  ci: string;
  telefono?: string;
  email?: string;
  estado: string;
}

export interface ReporteCliente {
  cliente_id: string;
  cliente?: ClienteInfoReporte;
  num_prestamos: number;
  prestamos_activos: number;
  total_prestado: number;
  total_pagado: number;
  saldo_total: number;
  mora_total: number;
  cuotas_vencidas: number;
  elegible_nuevo_prestamo: boolean;
  motivo_inelegible?: string;
  prestamos: PrestamoResumen[];
}

@Injectable({ providedIn: 'root' })
export class ReportService {
  private http = inject(HttpClient);
  private base = '/api/reports';

  dashboard(): Observable<DashboardData> {
    return this.http.get<DashboardData>(`${this.base}/dashboard`);
  }

  overdue(limit = 50): Observable<{ total: number; items: CuotaVencida[] }> {
    return this.http.get<{ total: number; items: CuotaVencida[] }>(`${this.base}/overdue`, {
      params: { limit: String(limit) },
    });
  }

  daily(date: string): Observable<ReporteDiario> {
    return this.http.get<ReporteDiario>(`${this.base}/daily`, { params: { date } });
  }

  monthly(year: number, month: number): Observable<ReporteMensual> {
    return this.http.get<ReporteMensual>(`${this.base}/monthly`, {
      params: { year: String(year), month: String(month) },
    });
  }

  /** Perfil crediticio consolidado del cliente (resumen + elegibilidad). */
  clientReport(id: string): Observable<ReporteCliente> {
    return this.http.get<ReporteCliente>(`${this.base}/clients/${id}`);
  }

  /**
   * Descarga un reporte en el formato dado como blob. Pasa por el interceptor
   * que agrega el Bearer token (el gateway lo exige). El nombre de archivo lo
   * arma el llamador para no depender de Content-Disposition vía CORS.
   */
  export(path: 'daily' | 'monthly' | 'overdue' | 'dashboard', format: ExportFormat, params: Record<string, string> = {}): Observable<Blob> {
    return this.http.get(`${this.base}/${path}`, {
      params: { ...params, format },
      responseType: 'blob',
    });
  }
}

export type ExportFormat = 'csv' | 'xlsx' | 'pdf';
