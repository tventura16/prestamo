import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { ReportService, DashboardData } from '../../core/services/report.service';

@Component({
  selector: 'app-dashboard',
  imports: [CommonModule, CurrencyPipe],
  template: `
    <h2>Dashboard</h2>

    @if (loading()) {
      <p class="hint">Cargando...</p>
    } @else if (error()) {
      <p class="err">Error: {{ error() }}</p>
    } @else if (data(); as d) {
      <div class="cards">
        <div class="card">
          <span class="label">Préstamos activos</span>
          <span class="value">{{ d.prestamos_activos }}</span>
        </div>
        <div class="card">
          <span class="label">Préstamos en mora</span>
          <span class="value" [class.warn]="d.prestamos_en_mora > 0">{{ d.prestamos_en_mora }}</span>
        </div>
        <div class="card">
          <span class="label">Clientes activos</span>
          <span class="value">{{ d.clientes_activos }}</span>
        </div>
        <div class="card">
          <span class="label">Cuotas vencidas</span>
          <span class="value" [class.warn]="d.cuotas_vencidas > 0">{{ d.cuotas_vencidas }}</span>
        </div>
        <div class="card">
          <span class="label">Ingresos del mes</span>
          <span class="value">{{ d.ingresos_mes | currency:'BOB':'symbol-narrow':'1.2-2' }}</span>
        </div>
        <div class="card">
          <span class="label">Ingresos de hoy</span>
          <span class="value">{{ d.ingresos_hoy | currency:'BOB':'symbol-narrow':'1.2-2' }}</span>
        </div>
        <div class="card big">
          <span class="label">Cartera por cobrar</span>
          <span class="value">{{ d.cartera_outstanding | currency:'BOB':'symbol-narrow':'1.2-2' }}</span>
        </div>
      </div>
    }
  `,
  styles: [`
    h2 { margin-top: 0; color: #2d3748; }
    .cards {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 16px;
    }
    .card {
      background: white;
      padding: 20px;
      border-radius: 8px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .card.big { grid-column: span 2; }
    .label { color: #718096; font-size: 13px; }
    .value { font-size: 28px; font-weight: 600; color: #1a365d; }
    .value.warn { color: #c53030; }
    .hint { color: #718096; }
    .err { color: #c53030; background: #fff5f5; padding: 12px; border-radius: 6px; }
  `],
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
