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
}
