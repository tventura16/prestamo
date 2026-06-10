import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { forkJoin } from 'rxjs';
import { ClienteService, Cliente } from '../../core/services/client.service';
import { ReportService, ReporteCliente } from '../../core/services/report.service';
import { PaymentService, Pago } from '../../core/services/payment.service';

@Component({
  selector: 'app-cliente-detail',
  imports: [CommonModule, CurrencyPipe, RouterLink],
  template: `
    <a routerLink="/clientes" class="text-sm text-navy-light hover:underline">← Clientes</a>

    @if (loading()) { <p class="mt-3 text-muted">Cargando historial...</p> }
    @if (error()) { <p class="mt-3 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ error() }}</p> }

    @if (perfil(); as p) {
      <header class="mb-4 mt-3 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 class="my-1 text-xl font-semibold text-ink">{{ p.cliente?.nombres }} {{ p.cliente?.apellidos }}</h2>
          <p class="m-0 text-sm text-muted">
            CI {{ p.cliente?.ci }}
            @if (p.cliente?.telefono) { · {{ p.cliente?.telefono }} }
            @if (p.cliente?.email) { · {{ p.cliente?.email }} }
            · <span class="rounded-full px-2.5 py-0.5 text-xs font-medium capitalize" [class]="badge(p.cliente?.estado || '')">{{ p.cliente?.estado }}</span>
          </p>
        </div>
        <!-- Veredicto de elegibilidad para un nuevo préstamo (regla §7). -->
        <div class="flex items-center gap-2 rounded-lg px-3.5 py-2 text-sm font-semibold"
             [class]="p.elegible_nuevo_prestamo ? 'bg-green-100 text-green-800' : 'bg-amber-100 text-amber-800'">
          <span class="inline-block h-2 w-2 rounded-full" [class]="p.elegible_nuevo_prestamo ? 'bg-green-600' : 'bg-amber-500'"></span>
          @if (p.elegible_nuevo_prestamo) { Elegible para nuevo préstamo }
          @else { No elegible — {{ p.motivo_inelegible }} }
        </div>
      </header>

      <!-- Resumen crediticio -->
      <div class="mb-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-7">
        <div class="flex flex-col gap-1 rounded-lg bg-white p-3.5 shadow-sm"><span class="text-xs text-muted">Préstamos</span><b class="text-lg text-ink">{{ p.num_prestamos }}</b></div>
        <div class="flex flex-col gap-1 rounded-lg bg-white p-3.5 shadow-sm"><span class="text-xs text-muted">Activos</span><b class="text-lg text-ink">{{ p.prestamos_activos }}</b></div>
        <div class="flex flex-col gap-1 rounded-lg bg-white p-3.5 shadow-sm"><span class="text-xs text-muted">Total prestado</span><b class="text-lg text-ink">{{ p.total_prestado | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></div>
        <div class="flex flex-col gap-1 rounded-lg bg-white p-3.5 shadow-sm"><span class="text-xs text-muted">Total pagado</span><b class="text-lg text-green-700">{{ p.total_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></div>
        <div class="flex flex-col gap-1 rounded-lg bg-white p-3.5 shadow-sm"><span class="text-xs text-muted">Saldo total</span><b class="text-lg text-ink">{{ p.saldo_total | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></div>
        <div class="flex flex-col gap-1 rounded-lg bg-white p-3.5 shadow-sm"><span class="text-xs text-muted">Mora total</span><b class="text-lg" [class]="p.mora_total > 0 ? 'text-red-600' : 'text-ink'">{{ p.mora_total | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></div>
        <div class="flex flex-col gap-1 rounded-lg bg-white p-3.5 shadow-sm"><span class="text-xs text-muted">Cuotas vencidas</span><b class="text-lg" [class]="p.cuotas_vencidas > 0 ? 'text-red-600' : 'text-ink'">{{ p.cuotas_vencidas }}</b></div>
      </div>

      <!-- Préstamos del cliente -->
      <section class="mb-4 rounded-lg bg-white p-4 shadow-sm">
        <h3 class="mb-3 text-base font-semibold text-ink">Préstamos</h3>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[760px] border-collapse text-sm">
            <thead>
              <tr class="border-b border-slate-200 bg-slate-50 text-left text-slate-600">
                <th class="px-3 py-2 font-semibold">Préstamo</th><th class="px-3 py-2 font-semibold">Estado</th><th class="px-3 py-2 text-right font-semibold">Monto</th>
                <th class="px-3 py-2 text-center font-semibold">Cuotas</th><th class="px-3 py-2 text-center font-semibold">Vencidas</th>
                <th class="px-3 py-2 text-right font-semibold">Saldo</th><th class="px-3 py-2 text-right font-semibold">Pagado</th><th class="px-3 py-2 font-semibold">Solicitud</th><th class="px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              @for (l of p.prestamos; track l.id) {
                <tr class="border-b border-slate-100 last:border-0">
                  <td class="px-3 py-2"><code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">{{ l.id | slice:0:8 }}</code></td>
                  <td class="px-3 py-2"><span class="rounded-full px-2.5 py-0.5 text-xs font-medium capitalize" [class]="badge(l.estado)">{{ l.estado }}</span></td>
                  <td class="px-3 py-2 text-right">{{ l.monto_aprobado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="px-3 py-2 text-center">{{ l.cuotas_pagadas }}/{{ l.num_cuotas }}</td>
                  <td class="px-3 py-2 text-center" [class.text-red-600]="l.cuotas_vencidas > 0" [class.font-bold]="l.cuotas_vencidas > 0">{{ l.cuotas_vencidas }}</td>
                  <td class="px-3 py-2 text-right">{{ l.saldo_pendiente | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="px-3 py-2 text-right text-green-700">{{ l.total_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="px-3 py-2 text-muted">{{ l.fecha_solicitud | slice:0:10 }}</td>
                  <td class="px-3 py-2"><a [routerLink]="['/prestamos', l.id]" class="text-navy-light hover:underline">ver</a></td>
                </tr>
              } @empty {
                <tr><td colspan="9" class="px-3 py-6 text-center text-muted">Sin préstamos</td></tr>
              }
            </tbody>
          </table>
        </div>
      </section>

      <!-- Historial de pagos -->
      <section class="mb-4 rounded-lg bg-white p-4 shadow-sm">
        <h3 class="mb-3 text-base font-semibold text-ink">Historial de pagos <span class="text-sm font-normal text-muted">({{ pagos().length }})</span></h3>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[720px] border-collapse text-sm">
            <thead>
              <tr class="border-b border-slate-200 bg-slate-50 text-left text-slate-600">
                <th class="px-3 py-2 font-semibold">Recibo</th><th class="px-3 py-2 font-semibold">Fecha</th><th class="px-3 py-2 text-right font-semibold">Monto</th>
                <th class="px-3 py-2 text-right font-semibold">Capital</th><th class="px-3 py-2 text-right font-semibold">Interés</th><th class="px-3 py-2 text-right font-semibold">Mora</th>
                <th class="px-3 py-2 font-semibold">Método</th><th class="px-3 py-2 font-semibold">Estado</th>
              </tr>
            </thead>
            <tbody>
              @for (pg of pagos(); track pg.id) {
                <tr class="border-b border-slate-100 last:border-0" [class.opacity-55]="pg.anulado">
                  <td class="px-3 py-2 font-semibold">{{ pg.numero_recibo }}</td>
                  <td class="px-3 py-2 text-muted">{{ pg.fecha_pago | slice:0:10 }}</td>
                  <td class="px-3 py-2 text-right font-semibold">{{ pg.monto_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="px-3 py-2 text-right">{{ pg.capital_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="px-3 py-2 text-right">{{ pg.interes_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="px-3 py-2 text-right">{{ pg.mora_pagada | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="px-3 py-2 capitalize">{{ pg.metodo_pago }}</td>
                  <td class="px-3 py-2">
                    @if (pg.anulado) { <span class="rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-800">anulado</span> }
                    @else { <span class="rounded-full px-2.5 py-0.5 text-xs font-medium capitalize" [class]="badge(pg.tipo)">{{ pg.tipo }}</span> }
                  </td>
                </tr>
              } @empty {
                <tr><td colspan="8" class="px-3 py-6 text-center text-muted">Sin pagos registrados</td></tr>
              }
            </tbody>
          </table>
        </div>
      </section>
    }
  `,
})
export class ClienteDetail implements OnInit {
  private route = inject(ActivatedRoute);
  private clientes = inject(ClienteService);
  private reports = inject(ReportService);
  private pagosSvc = inject(PaymentService);

  perfil = signal<ReporteCliente | null>(null);
  cliente = signal<Cliente | null>(null);
  pagos = signal<Pago[]>([]);
  loading = signal(false);
  error = signal<string | null>(null);

  // Color de badge por estado (cliente, préstamo, cuota o tipo de pago).
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
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) {
      this.error.set('Cliente no especificado');
      return;
    }
    this.loading.set(true);
    // El perfil del report-service ya trae los datos del cliente; client.get
    // es respaldo por si el reporte no los incluyera. Pagos en paralelo.
    forkJoin({
      perfil: this.reports.clientReport(id),
      cliente: this.clientes.get(id),
      pagos: this.pagosSvc.list({ cliente_id: id, limit: 100 }),
    }).subscribe({
      next: ({ perfil, cliente, pagos }) => {
        if (!perfil.cliente) {
          perfil.cliente = {
            nombres: cliente.nombres, apellidos: cliente.apellidos, ci: cliente.ci,
            telefono: cliente.telefono, email: cliente.email, estado: cliente.estado,
          };
        }
        this.perfil.set(perfil);
        this.cliente.set(cliente);
        this.pagos.set(pagos.items);
        this.loading.set(false);
      },
      error: e => { this.error.set(e.error?.error || e.message); this.loading.set(false); },
    });
  }
}
