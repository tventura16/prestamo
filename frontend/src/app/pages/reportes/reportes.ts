import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  ReportService, ReporteDiario, ReporteMensual, CuotaVencida,
} from '../../core/services/report.service';

@Component({
  selector: 'app-reportes',
  imports: [CommonModule, FormsModule, CurrencyPipe],
  template: `
    <h2>Reportes</h2>
    @if (error()) { <p class="err">{{ error() }}</p> }

    <div class="grid">
      <!-- ─── Reporte diario ─── -->
      <section class="card">
        <header><h3>Reporte diario</h3>
          <input type="date" [(ngModel)]="fecha" (change)="loadDiario()" />
        </header>
        @if (loadingDiario()) { <p class="hint">Cargando...</p> }
        @if (diario(); as d) {
          <dl>
            <div><dt>Ingresos</dt><dd class="money">{{ d.ingresos | currency:'BOB':'symbol-narrow':'1.2-2' }}</dd></div>
            <div><dt>Mora cobrada</dt><dd>{{ d.mora_cobrada | currency:'BOB':'symbol-narrow':'1.2-2' }}</dd></div>
            <div><dt>Pagos recibidos</dt><dd>{{ d.pagos_recibidos }}</dd></div>
            <div><dt>Préstamos nuevos</dt><dd>{{ d.prestamos_nuevos }}</dd></div>
            <div><dt>Clientes nuevos</dt><dd>{{ d.clientes_nuevos }}</dd></div>
          </dl>
        }
      </section>

      <!-- ─── Reporte mensual ─── -->
      <section class="card">
        <header><h3>Reporte mensual</h3>
          <span class="period">
            <input type="number" min="2000" max="2100" [(ngModel)]="anio" (change)="loadMensual()" />
            <select [(ngModel)]="mes" (change)="loadMensual()">
              @for (m of meses; track m.v) { <option [value]="m.v">{{ m.n }}</option> }
            </select>
          </span>
        </header>
        @if (loadingMensual()) { <p class="hint">Cargando...</p> }
        @if (mensual(); as m) {
          <dl>
            <div><dt>Ingresos</dt><dd class="money">{{ m.ingresos | currency:'BOB':'symbol-narrow':'1.2-2' }}</dd></div>
            <div><dt>Intereses pagados</dt><dd>{{ m.intereses_pagados | currency:'BOB':'symbol-narrow':'1.2-2' }}</dd></div>
            <div><dt>Mora cobrada</dt><dd>{{ m.mora_cobrada | currency:'BOB':'symbol-narrow':'1.2-2' }}</dd></div>
            <div><dt>Pagos recibidos</dt><dd>{{ m.pagos_recibidos }}</dd></div>
            <div><dt>Préstamos nuevos</dt><dd>{{ m.prestamos_nuevos }}</dd></div>
            <div><dt>Clientes nuevos</dt><dd>{{ m.clientes_nuevos }}</dd></div>
          </dl>
        }
      </section>
    </div>

    <!-- ─── Cuotas vencidas (mora) ─── -->
    <section class="card full">
      <header><h3>Cuotas vencidas</h3>
        <span class="hint">{{ totalVencidas() }} en mora</span>
      </header>
      @if (loadingVencidas()) { <p class="hint">Cargando...</p> }
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Cliente</th><th>Préstamo</th><th class="c">Cuota #</th>
              <th>Vencimiento</th><th class="r">Días</th>
              <th class="r">Saldo</th><th class="r">Mora</th><th>Estado</th>
            </tr>
          </thead>
          <tbody>
            @for (c of vencidas(); track c.cuota_id) {
              <tr>
                <td><code>{{ c.cliente_id | slice:0:8 }}</code></td>
                <td><code>{{ c.prestamo_id | slice:0:8 }}</code></td>
                <td class="c">{{ c.numero }}</td>
                <td class="muted">{{ c.fecha_vencimiento | slice:0:10 }}</td>
                <td class="r" [class.danger]="c.dias_vencidos > 30">{{ c.dias_vencidos }}</td>
                <td class="r">{{ c.saldo_pendiente | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td class="r money">{{ c.mora_acumulada | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td><span class="badge">{{ c.estado }}</span></td>
              </tr>
            } @empty {
              <tr><td colspan="8" class="muted center">Sin cuotas vencidas 🎉</td></tr>
            }
          </tbody>
        </table>
      </div>
    </section>
  `,
  styles: [`
    h2, h3 { color: #2d3748; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 16px; margin-bottom: 16px; }
    .card { background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); padding: 16px; }
    .card.full { padding: 16px; }
    .card header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; gap: 8px; }
    .card h3 { margin: 0; font-size: 15px; }
    .period { display: flex; gap: 6px; }
    input, select { padding: 5px 8px; border: 1px solid #cbd5e0; border-radius: 6px; font-size: 13px; }
    input[type=number] { width: 80px; }
    dl { margin: 0; display: grid; grid-template-columns: 1fr; gap: 4px; }
    dl div { display: flex; justify-content: space-between; padding: 6px 0; border-bottom: 1px solid #edf2f7; }
    dt { color: #718096; font-size: 13px; }
    dd { margin: 0; font-weight: 600; color: #2d3748; }
    .money { color: #2f855a; }
    .table-wrap { overflow: hidden; border-radius: 6px; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    th, td { padding: 8px 12px; text-align: left; }
    th { background: #f7fafc; color: #4a5568; font-weight: 600; border-bottom: 1px solid #e2e8f0; }
    td { border-bottom: 1px solid #edf2f7; }
    .r { text-align: right; } .c { text-align: center; }
    .danger { color: #c53030; font-weight: 700; }
    code { background: #edf2f7; padding: 2px 6px; border-radius: 4px; font-size: 11px; }
    .badge { padding: 2px 10px; border-radius: 12px; font-size: 12px; background: #fed7d7; color: #822727; }
    .muted { color: #718096; } .center { text-align: center; }
    .hint { color: #718096; font-size: 13px; }
    .err { color: #c53030; background: #fff5f5; padding: 10px; border-radius: 6px; }
  `],
})
export class Reportes implements OnInit {
  private svc = inject(ReportService);

  private today = new Date();
  fecha = this.today.toISOString().slice(0, 10);
  anio = this.today.getFullYear();
  mes = this.today.getMonth() + 1;
  readonly meses = [
    { v: 1, n: 'Enero' }, { v: 2, n: 'Febrero' }, { v: 3, n: 'Marzo' },
    { v: 4, n: 'Abril' }, { v: 5, n: 'Mayo' }, { v: 6, n: 'Junio' },
    { v: 7, n: 'Julio' }, { v: 8, n: 'Agosto' }, { v: 9, n: 'Septiembre' },
    { v: 10, n: 'Octubre' }, { v: 11, n: 'Noviembre' }, { v: 12, n: 'Diciembre' },
  ];

  diario = signal<ReporteDiario | null>(null);
  mensual = signal<ReporteMensual | null>(null);
  vencidas = signal<CuotaVencida[]>([]);
  totalVencidas = signal(0);
  loadingDiario = signal(false);
  loadingMensual = signal(false);
  loadingVencidas = signal(false);
  error = signal<string | null>(null);

  ngOnInit() {
    this.loadDiario();
    this.loadMensual();
    this.loadVencidas();
  }

  loadDiario() {
    this.loadingDiario.set(true);
    this.svc.daily(this.fecha).subscribe({
      next: d => { this.diario.set(d); this.loadingDiario.set(false); },
      error: e => { this.error.set(e.error?.error || e.message); this.loadingDiario.set(false); },
    });
  }

  loadMensual() {
    this.loadingMensual.set(true);
    this.svc.monthly(Number(this.anio), Number(this.mes)).subscribe({
      next: m => { this.mensual.set(m); this.loadingMensual.set(false); },
      error: e => { this.error.set(e.error?.error || e.message); this.loadingMensual.set(false); },
    });
  }

  loadVencidas() {
    this.loadingVencidas.set(true);
    this.svc.overdue(100).subscribe({
      next: r => { this.vencidas.set(r.items); this.totalVencidas.set(r.total); this.loadingVencidas.set(false); },
      error: e => { this.error.set(e.error?.error || e.message); this.loadingVencidas.set(false); },
    });
  }
}
