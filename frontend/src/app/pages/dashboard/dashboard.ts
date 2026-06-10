import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { ReportService, DashboardData } from '../../core/services/report.service';

@Component({
  selector: 'app-dashboard',
  imports: [CommonModule, CurrencyPipe],
  template: `
    <h2 class="mb-4 mt-0 text-xl font-semibold text-ink">Dashboard</h2>

    @if (loading()) {
      <p class="text-muted">Cargando...</p>
    } @else if (error()) {
      <p class="rounded-md bg-red-50 p-3 text-red-600">Error: {{ error() }}</p>
    } @else if (data(); as d) {
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div class="flex flex-col gap-2 rounded-lg bg-white p-5 shadow-sm">
          <span class="text-sm text-muted">Préstamos activos</span>
          <span class="text-3xl font-semibold text-navy">{{ d.prestamos_activos }}</span>
        </div>
        <div class="flex flex-col gap-2 rounded-lg bg-white p-5 shadow-sm">
          <span class="text-sm text-muted">Préstamos en mora</span>
          <span class="text-3xl font-semibold" [class]="d.prestamos_en_mora > 0 ? 'text-red-600' : 'text-navy'">{{ d.prestamos_en_mora }}</span>
        </div>
        <div class="flex flex-col gap-2 rounded-lg bg-white p-5 shadow-sm">
          <span class="text-sm text-muted">Clientes activos</span>
          <span class="text-3xl font-semibold text-navy">{{ d.clientes_activos }}</span>
        </div>
        <div class="flex flex-col gap-2 rounded-lg bg-white p-5 shadow-sm">
          <span class="text-sm text-muted">Cuotas vencidas</span>
          <span class="text-3xl font-semibold" [class]="d.cuotas_vencidas > 0 ? 'text-red-600' : 'text-navy'">{{ d.cuotas_vencidas }}</span>
        </div>
        <div class="flex flex-col gap-2 rounded-lg bg-white p-5 shadow-sm">
          <span class="text-sm text-muted">Ingresos del mes</span>
          <span class="text-3xl font-semibold text-navy">{{ d.ingresos_mes | currency:'BOB':'symbol-narrow':'1.2-2' }}</span>
        </div>
        <div class="flex flex-col gap-2 rounded-lg bg-white p-5 shadow-sm">
          <span class="text-sm text-muted">Ingresos de hoy</span>
          <span class="text-3xl font-semibold text-navy">{{ d.ingresos_hoy | currency:'BOB':'symbol-narrow':'1.2-2' }}</span>
        </div>
        <div class="flex flex-col gap-2 rounded-lg bg-navy p-5 text-white shadow-sm sm:col-span-2 lg:col-span-3">
          <span class="text-sm text-slate-300">Cartera por cobrar</span>
          <span class="text-3xl font-semibold">{{ d.cartera_outstanding | currency:'BOB':'symbol-narrow':'1.2-2' }}</span>
        </div>
      </div>
    }
  `,
})
export class Dashboard implements OnInit {
  private reportSvc = inject(ReportService);

  data = signal<DashboardData | null>(null);
  loading = signal(true);
  error = signal<string | null>(null);

  ngOnInit() {
    this.reportSvc.dashboard().subscribe({
      next: d => { this.data.set(d); this.loading.set(false); },
      error: e => { this.error.set(e.message || 'Error desconocido'); this.loading.set(false); },
    });
  }
}
