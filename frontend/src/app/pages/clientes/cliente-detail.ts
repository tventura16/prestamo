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
    <a routerLink="/clientes" class="back">← Clientes</a>

    @if (loading()) { <p class="hint">Cargando historial...</p> }
    @if (error()) { <p class="err">{{ error() }}</p> }

    @if (perfil(); as p) {
      <header class="head">
        <div>
          <h2>{{ p.cliente?.nombres }} {{ p.cliente?.apellidos }}</h2>
          <p class="sub">
            CI {{ p.cliente?.ci }}
            @if (p.cliente?.telefono) { · {{ p.cliente?.telefono }} }
            @if (p.cliente?.email) { · {{ p.cliente?.email }} }
            · <span class="badge" [class]="'st-' + (p.cliente?.estado || '')">{{ p.cliente?.estado }}</span>
          </p>
        </div>
        <!-- Veredicto de elegibilidad para un nuevo préstamo (regla §7). -->
        <div class="elig" [class.ok]="p.elegible_nuevo_prestamo" [class.no]="!p.elegible_nuevo_prestamo">
          @if (p.elegible_nuevo_prestamo) {
            <span class="dot"></span> Elegible para nuevo préstamo
          } @else {
            <span class="dot"></span> No elegible — {{ p.motivo_inelegible }}
          }
        </div>
      </header>

      <!-- Resumen crediticio -->
      <div class="cards">
        <div class="kpi"><span>Préstamos</span><b>{{ p.num_prestamos }}</b></div>
        <div class="kpi"><span>Activos</span><b>{{ p.prestamos_activos }}</b></div>
        <div class="kpi"><span>Total prestado</span><b>{{ p.total_prestado | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></div>
        <div class="kpi"><span>Total pagado</span><b class="money">{{ p.total_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></div>
        <div class="kpi"><span>Saldo total</span><b>{{ p.saldo_total | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></div>
        <div class="kpi"><span>Mora total</span><b [class.danger]="p.mora_total > 0">{{ p.mora_total | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></div>
        <div class="kpi"><span>Cuotas vencidas</span><b [class.danger]="p.cuotas_vencidas > 0">{{ p.cuotas_vencidas }}</b></div>
      </div>

      <!-- Préstamos del cliente -->
      <section class="card">
        <h3>Préstamos</h3>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Préstamo</th><th>Estado</th><th class="r">Monto</th>
                <th class="c">Cuotas</th><th class="c">Vencidas</th>
                <th class="r">Saldo</th><th class="r">Pagado</th><th>Solicitud</th><th></th>
              </tr>
            </thead>
            <tbody>
              @for (l of p.prestamos; track l.id) {
                <tr>
                  <td><code>{{ l.id | slice:0:8 }}</code></td>
                  <td><span class="badge" [class]="'st-' + l.estado">{{ l.estado }}</span></td>
                  <td class="r">{{ l.monto_aprobado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="c">{{ l.cuotas_pagadas }}/{{ l.num_cuotas }}</td>
                  <td class="c" [class.danger]="l.cuotas_vencidas > 0">{{ l.cuotas_vencidas }}</td>
                  <td class="r">{{ l.saldo_pendiente | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="r money">{{ l.total_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="muted">{{ l.fecha_solicitud | slice:0:10 }}</td>
                  <td><a [routerLink]="['/prestamos', l.id]" class="link">ver</a></td>
                </tr>
              } @empty {
                <tr><td colspan="9" class="muted center">Sin préstamos</td></tr>
              }
            </tbody>
          </table>
        </div>
      </section>

      <!-- Historial de pagos -->
      <section class="card">
        <h3>Historial de pagos <span class="hint">({{ pagos().length }})</span></h3>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Recibo</th><th>Fecha</th><th class="r">Monto</th>
                <th class="r">Capital</th><th class="r">Interés</th><th class="r">Mora</th>
                <th>Método</th><th>Estado</th>
              </tr>
            </thead>
            <tbody>
              @for (pg of pagos(); track pg.id) {
                <tr [class.anulado]="pg.anulado">
                  <td><b>{{ pg.numero_recibo }}</b></td>
                  <td class="muted">{{ pg.fecha_pago | slice:0:10 }}</td>
                  <td class="r"><b>{{ pg.monto_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></td>
                  <td class="r">{{ pg.capital_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="r">{{ pg.interes_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td class="r">{{ pg.mora_pagada | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                  <td>{{ pg.metodo_pago }}</td>
                  <td>
                    @if (pg.anulado) { <span class="badge st-anulado">anulado</span> }
                    @else { <span class="badge st-activo">{{ pg.tipo }}</span> }
                  </td>
                </tr>
              } @empty {
                <tr><td colspan="8" class="muted center">Sin pagos registrados</td></tr>
              }
            </tbody>
          </table>
        </div>
      </section>
    }
  `,
  styles: [`
    .back { color: #2c5282; text-decoration: none; font-size: 13px; }
    .back:hover { text-decoration: underline; }
    .head { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; margin: 12px 0 16px; flex-wrap: wrap; }
    h2 { color: #2d3748; margin: 4px 0; }
    h3 { color: #2d3748; font-size: 15px; margin: 0 0 12px; }
    .sub { color: #718096; font-size: 13px; margin: 0; }
    .elig { padding: 8px 14px; border-radius: 8px; font-size: 13px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
    .elig .dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
    .elig.ok { background: #c6f6d5; color: #22543d; } .elig.ok .dot { background: #38a169; }
    .elig.no { background: #fefcbf; color: #744210; } .elig.no .dot { background: #d69e2e; }
    .cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px; margin-bottom: 16px; }
    .kpi { background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); padding: 12px 14px; display: flex; flex-direction: column; gap: 4px; }
    .kpi span { color: #718096; font-size: 12px; } .kpi b { font-size: 18px; color: #2d3748; }
    .card { background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); padding: 16px; margin-bottom: 16px; }
    .table-wrap { overflow: hidden; border-radius: 6px; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    th, td { padding: 8px 12px; text-align: left; }
    th { background: #f7fafc; color: #4a5568; font-weight: 600; border-bottom: 1px solid #e2e8f0; }
    td { border-bottom: 1px solid #edf2f7; }
    .r { text-align: right; } .c { text-align: center; }
    .money { color: #2f855a; } .danger { color: #c53030; font-weight: 700; }
    code { background: #edf2f7; padding: 2px 6px; border-radius: 4px; font-size: 11px; }
    .link { color: #2c5282; text-decoration: none; } .link:hover { text-decoration: underline; }
    .badge { padding: 2px 10px; border-radius: 12px; font-size: 12px; text-transform: capitalize; }
    .st-activo { background: #c6f6d5; color: #22543d; }
    .st-mora, .st-anulado, .st-bloqueado { background: #fed7d7; color: #822727; }
    .st-finalizado { background: #e2e8f0; color: #2d3748; }
    .st-pendiente, .st-inactivo { background: #feebc8; color: #7b341e; }
    .st-rechazado { background: #fed7d7; color: #822727; }
    tr.anulado { opacity: 0.55; }
    tr.anulado td:not(:last-child) { text-decoration: line-through; }
    .muted { color: #718096; } .center { text-align: center; }
    .hint { color: #718096; font-size: 13px; font-weight: 400; }
    .err { color: #c53030; background: #fff5f5; padding: 10px; border-radius: 6px; }
  `],
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
