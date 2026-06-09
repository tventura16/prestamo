import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import {
  ReportService, ReporteDiario, ReporteMensual, CuotaVencida, ExportFormat,
} from '../../core/services/report.service';

type ReportPath = 'daily' | 'monthly' | 'overdue';

@Component({
  selector: 'app-reportes',
  imports: [CommonModule, FormsModule, CurrencyPipe],
  template: `
    <h2 class="text-xl font-semibold text-ink">Reportes</h2>
    @if (error()) { <p class="mt-3 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ error() }}</p> }

    <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
      <!-- ─── Reporte diario ─── -->
      <section class="rounded-lg bg-white p-4 shadow-sm">
        <header class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h3 class="text-base font-semibold text-ink">Reporte diario</h3>
          <div class="flex flex-wrap items-center gap-2">
            <input type="date" class="ui-input" [(ngModel)]="fecha" (change)="loadDiario()" />
            <ng-container [ngTemplateOutlet]="dl" [ngTemplateOutletContext]="{ path: 'daily' }"></ng-container>
          </div>
        </header>
        @if (loadingDiario()) { <p class="text-sm text-muted">Cargando...</p> }
        @if (diario(); as d) {
          <dl class="m-0">
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Ingresos</dt><dd class="m-0 font-semibold text-green-700">{{ d.ingresos | currency:'BOB':'symbol-narrow':'1.2-2' }}</dd></div>
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Mora cobrada</dt><dd class="m-0 font-semibold text-ink">{{ d.mora_cobrada | currency:'BOB':'symbol-narrow':'1.2-2' }}</dd></div>
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Pagos recibidos</dt><dd class="m-0 font-semibold text-ink">{{ d.pagos_recibidos }}</dd></div>
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Préstamos nuevos</dt><dd class="m-0 font-semibold text-ink">{{ d.prestamos_nuevos }}</dd></div>
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Clientes nuevos</dt><dd class="m-0 font-semibold text-ink">{{ d.clientes_nuevos }}</dd></div>
          </dl>
        }
      </section>

      <!-- ─── Reporte mensual ─── -->
      <section class="rounded-lg bg-white p-4 shadow-sm">
        <header class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h3 class="text-base font-semibold text-ink">Reporte mensual</h3>
          <div class="flex flex-wrap items-center gap-2">
            <span class="flex gap-1.5">
              <input type="number" min="2000" max="2100" class="ui-input w-20" [(ngModel)]="anio" (change)="loadMensual()" />
              <select class="ui-input" [(ngModel)]="mes" (change)="loadMensual()">
                @for (m of meses; track m.v) { <option [value]="m.v">{{ m.n }}</option> }
              </select>
            </span>
            <ng-container [ngTemplateOutlet]="dl" [ngTemplateOutletContext]="{ path: 'monthly' }"></ng-container>
          </div>
        </header>
        @if (loadingMensual()) { <p class="text-sm text-muted">Cargando...</p> }
        @if (mensual(); as m) {
          <dl class="m-0">
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Ingresos</dt><dd class="m-0 font-semibold text-green-700">{{ m.ingresos | currency:'BOB':'symbol-narrow':'1.2-2' }}</dd></div>
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Intereses pagados</dt><dd class="m-0 font-semibold text-ink">{{ m.intereses_pagados | currency:'BOB':'symbol-narrow':'1.2-2' }}</dd></div>
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Mora cobrada</dt><dd class="m-0 font-semibold text-ink">{{ m.mora_cobrada | currency:'BOB':'symbol-narrow':'1.2-2' }}</dd></div>
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Pagos recibidos</dt><dd class="m-0 font-semibold text-ink">{{ m.pagos_recibidos }}</dd></div>
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Préstamos nuevos</dt><dd class="m-0 font-semibold text-ink">{{ m.prestamos_nuevos }}</dd></div>
            <div class="flex justify-between border-b border-slate-100 py-1.5"><dt class="text-muted">Clientes nuevos</dt><dd class="m-0 font-semibold text-ink">{{ m.clientes_nuevos }}</dd></div>
          </dl>
        }
      </section>
    </div>

    <!-- ─── Cuotas vencidas (mora) ─── -->
    <section class="mt-4">
      <header class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h3 class="text-base font-semibold text-ink">Cuotas vencidas</h3>
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-sm text-muted">{{ totalVencidas() }} en mora</span>
          <ng-container [ngTemplateOutlet]="dl" [ngTemplateOutletContext]="{ path: 'overdue' }"></ng-container>
        </div>
      </header>
      @if (loadingVencidas()) { <p class="text-sm text-muted">Cargando...</p> }
      <div class="overflow-x-auto rounded-lg bg-white shadow-sm">
        <table class="w-full min-w-[720px] border-collapse text-sm">
          <thead>
            <tr class="border-b border-slate-200 bg-slate-50 text-left text-slate-600">
              <th class="px-3 py-2 font-semibold">Cliente</th><th class="px-3 py-2 font-semibold">Préstamo</th><th class="px-3 py-2 text-center font-semibold">Cuota #</th>
              <th class="px-3 py-2 font-semibold">Vencimiento</th><th class="px-3 py-2 text-right font-semibold">Días</th>
              <th class="px-3 py-2 text-right font-semibold">Saldo</th><th class="px-3 py-2 text-right font-semibold">Mora</th><th class="px-3 py-2 font-semibold">Estado</th>
            </tr>
          </thead>
          <tbody>
            @for (c of vencidas(); track c.cuota_id) {
              <tr class="border-b border-slate-100 last:border-0">
                <td class="px-3 py-2"><code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">{{ c.cliente_id | slice:0:8 }}</code></td>
                <td class="px-3 py-2"><code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">{{ c.prestamo_id | slice:0:8 }}</code></td>
                <td class="px-3 py-2 text-center">{{ c.numero }}</td>
                <td class="px-3 py-2 text-muted">{{ c.fecha_vencimiento | slice:0:10 }}</td>
                <td class="px-3 py-2 text-right" [class.text-red-600]="c.dias_vencidos > 30" [class.font-bold]="c.dias_vencidos > 30">{{ c.dias_vencidos }}</td>
                <td class="px-3 py-2 text-right">{{ c.saldo_pendiente | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td class="px-3 py-2 text-right font-semibold text-green-700">{{ c.mora_acumulada | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td class="px-3 py-2"><span class="rounded-full px-2.5 py-0.5 text-xs font-medium capitalize" [class]="badge(c.estado)">{{ c.estado }}</span></td>
              </tr>
            } @empty {
              <tr><td colspan="8" class="px-3 py-6 text-center text-muted">Sin cuotas vencidas 🎉</td></tr>
            }
          </tbody>
        </table>
      </div>
    </section>

    <!-- Grupo de botones de descarga reutilizable -->
    <ng-template #dl let-path="path">
      <span class="inline-flex flex-wrap items-center gap-1">
        <span class="text-xs uppercase tracking-wide text-slate-400">Exportar</span>
        <button class="rounded-md border border-slate-300 bg-slate-50 px-2.5 py-1 text-xs font-semibold text-navy-light hover:bg-sky-50 disabled:opacity-50" (click)="descargar(path, 'csv')" [disabled]="descargando() === path + ':csv'">CSV</button>
        <button class="rounded-md border border-slate-300 bg-slate-50 px-2.5 py-1 text-xs font-semibold text-navy-light hover:bg-sky-50 disabled:opacity-50" (click)="descargar(path, 'xlsx')" [disabled]="descargando() === path + ':xlsx'">Excel</button>
        <button class="rounded-md border border-slate-300 bg-slate-50 px-2.5 py-1 text-xs font-semibold text-navy-light hover:bg-sky-50 disabled:opacity-50" (click)="descargar(path, 'pdf')" [disabled]="descargando() === path + ':pdf'">PDF</button>
      </span>
    </ng-template>
  `,
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
  descargando = signal<string | null>(null);
  error = signal<string | null>(null);

  // Color de badge por estado de cuota (mismo vocabulario que cliente-detail).
  badge(estado: string): string {
    switch (estado) {
      case 'activo':
      case 'pagada':
      case 'total':
        return 'bg-green-100 text-green-800';
      case 'mora':
      case 'anulado':
      case 'bloqueado':
      case 'rechazado':
      case 'vencida':
        return 'bg-red-100 text-red-800';
      case 'finalizado':
        return 'bg-slate-200 text-slate-700';
      case 'pendiente':
      case 'inactivo':
      case 'parcial':
        return 'bg-orange-100 text-orange-800';
      default:
        return 'bg-slate-200 text-slate-600';
    }
  }

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

  // Descarga el reporte indicado con los parámetros del filtro vigente. El
  // nombre de archivo se arma aquí para no depender de Content-Disposition.
  descargar(path: ReportPath, format: ExportFormat) {
    const mm = String(this.mes).padStart(2, '0');
    const cfg: Record<ReportPath, { params: Record<string, string>; base: string }> = {
      daily: { params: { date: this.fecha }, base: `reporte-diario-${this.fecha}` },
      monthly: { params: { year: String(this.anio), month: String(this.mes) }, base: `reporte-mensual-${this.anio}-${mm}` },
      overdue: { params: { limit: '100' }, base: `cuotas-vencidas-${this.fecha}` },
    };
    const { params, base } = cfg[path];

    this.descargando.set(`${path}:${format}`);
    this.error.set(null);
    this.svc.export(path, format, params).subscribe({
      next: blob => {
        this.descargando.set(null);
        this.saveBlob(blob, `${base}.${format}`);
      },
      error: () => {
        this.descargando.set(null);
        this.error.set('No se pudo descargar el reporte');
      },
    });
  }

  private saveBlob(blob: Blob, filename: string) {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }
}
