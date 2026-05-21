import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';

import { LoanService, Prestamo, Cuota } from '../../core/services/loan.service';
import { PaymentService, MetodoPago } from '../../core/services/payment.service';
import { DocumentService } from '../../core/services/document.service';
import { KeycloakService } from '../../core/keycloak.service';

@Component({
  selector: 'app-prestamo-detail',
  imports: [CommonModule, CurrencyPipe, RouterLink, FormsModule],
  template: `
    <a routerLink="/prestamos" class="back">← volver</a>

    @if (loading()) { <p class="hint">Cargando...</p> }
    @if (error()) { <p class="err">{{ error() }}</p> }

    @if (prestamo(); as p) {
      <div class="card-info">
        <div class="title">
          <h2>Préstamo</h2>
          <span class="badge" [class]="'b-' + p.estado">{{ p.estado }}</span>
        </div>
        <div class="grid">
          <div><b>ID:</b> <code>{{ p.id }}</code></div>
          <div><b>Cliente:</b> <code>{{ p.cliente_id }}</code></div>
          <div><b>Monto solicitado:</b> {{ p.monto_solicitado | currency:'BOB':'symbol-narrow':'1.2-2' }}</div>
          <div><b>Monto aprobado:</b> {{ (p.monto_aprobado ?? 0) | currency:'BOB':'symbol-narrow':'1.2-2' }}</div>
          <div><b>Tasa:</b> {{ (p.tasa_interes * 100).toFixed(2) }}% ({{ p.tipo_interes }})</div>
          <div><b>Cuotas:</b> {{ p.num_cuotas }} ({{ p.frecuencia }})</div>
          <div><b>Solicitud:</b> {{ p.fecha_solicitud | slice:0:10 }}</div>
          <div><b>Desembolso:</b> {{ (p.fecha_desembolso | slice:0:10) || '—' }}</div>
        </div>
        @if (p.observaciones) {
          <p><b>Observaciones:</b> {{ p.observaciones }}</p>
        }

        <!-- Acciones disponibles según estado -->
        <div class="actions">
          @if (p.estado === 'pendiente') {
            <button class="btn success" (click)="openApprove()">✓ Aprobar</button>
            <button class="btn danger" (click)="openReject()">✗ Rechazar</button>
          }
          <button class="btn primary" (click)="generarContrato()" [disabled]="generando()">
            {{ generando() ? 'Generando...' : '📄 Contrato PDF' }}
          </button>
        </div>

        <!-- Form Aprobar -->
        @if (showApprove()) {
          <div class="inline-form">
            <h4>Aprobar préstamo</h4>
            <div class="row">
              <label>Monto aprobado (default = solicitado)
                <input type="number" min="1" step="0.01"
                       [(ngModel)]="approveForm.monto_aprobado" name="monto_a">
              </label>
              <label>Fecha desembolso
                <input type="date" [(ngModel)]="approveForm.fecha_desembolso" name="fecha_d">
              </label>
            </div>
            @if (approveError()) { <p class="err">{{ approveError() }}</p> }
            <div class="actions">
              <button class="btn" (click)="showApprove.set(false)">Cancelar</button>
              <button class="btn success" (click)="doApprove()" [disabled]="approving()">
                {{ approving() ? 'Aprobando...' : 'Confirmar aprobación' }}
              </button>
            </div>
          </div>
        }

        <!-- Form Rechazar -->
        @if (showReject()) {
          <div class="inline-form">
            <h4>Rechazar préstamo</h4>
            <label>Motivo *
              <input [(ngModel)]="rejectForm.observaciones" name="obs" required minlength="3">
            </label>
            @if (rejectError()) { <p class="err">{{ rejectError() }}</p> }
            <div class="actions">
              <button class="btn" (click)="showReject.set(false)">Cancelar</button>
              <button class="btn danger" (click)="doReject()"
                      [disabled]="rejecting() || !rejectForm.observaciones">
                {{ rejecting() ? 'Rechazando...' : 'Confirmar rechazo' }}
              </button>
            </div>
          </div>
        }
      </div>

      <h3>Plan de pagos</h3>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>#</th>
              <th>Vencimiento</th>
              <th class="r">Capital</th>
              <th class="r">Interés</th>
              <th class="r">Total</th>
              <th class="r">Saldo</th>
              <th>Estado</th>
              <th>Pago</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            @for (c of cuotas(); track c.id) {
              <tr [class.pagada]="c.estado === 'pagada'">
                <td>{{ c.numero }}</td>
                <td>{{ c.fecha_vencimiento | slice:0:10 }}</td>
                <td class="r">{{ c.capital | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td class="r">{{ c.interes | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td class="r"><b>{{ c.total | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></td>
                <td class="r">{{ c.saldo_pendiente | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td><span class="badge" [class]="'c-' + c.estado">{{ c.estado }}</span></td>
                <td class="muted">{{ (c.fecha_pago | slice:0:10) || '—' }}</td>
                <td>
                  @if (c.estado !== 'pagada' && p.estado === 'activo') {
                    <button class="link-btn" (click)="openPay(c)">💰 Pagar</button>
                  }
                </td>
              </tr>
            } @empty {
              <tr><td colspan="9" class="muted center">Sin cuotas (préstamo aún no aprobado)</td></tr>
            }
          </tbody>
        </table>
      </div>

      @if (cuotas().length > 0) {
        <div class="totals">
          <span>Capital: <b>{{ totalCapital() | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></span>
          <span>Interés: <b>{{ totalInteres() | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></span>
          <span>Total: <b>{{ totalGeneral() | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></span>
        </div>
      }

      <!-- Form Pagar Cuota -->
      @if (payCuota(); as pc) {
        <div class="inline-form sticky">
          <h4>Registrar pago — Cuota #{{ pc.numero }}</h4>
          <div class="row">
            <label>Monto a pagar *
              <input type="number" min="0.01" step="0.01" max="{{ pc.saldo_pendiente + pc.mora_acumulada }}"
                     [(ngModel)]="payForm.monto_pagado" name="monto_p" required>
            </label>
            <label>Método de pago *
              <select [(ngModel)]="payForm.metodo_pago" name="metodo_p" required>
                <option value="efectivo">Efectivo</option>
                <option value="transferencia">Transferencia</option>
                <option value="cheque">Cheque</option>
                <option value="tarjeta">Tarjeta</option>
                <option value="qr">QR</option>
              </select>
            </label>
          </div>
          <label class="full">Observaciones
            <input [(ngModel)]="payForm.observaciones" name="obs_p">
          </label>
          <p class="hint">
            Saldo cuota: <b>{{ pc.saldo_pendiente | currency:'BOB':'symbol-narrow':'1.2-2' }}</b>
            @if (pc.mora_acumulada > 0) {
              · Mora: <b>{{ pc.mora_acumulada | currency:'BOB':'symbol-narrow':'1.2-2' }}</b>
            }
          </p>
          @if (payError()) { <p class="err">{{ payError() }}</p> }
          <div class="actions">
            <button class="btn" (click)="payCuota.set(null)">Cancelar</button>
            <button class="btn primary" (click)="doPay()" [disabled]="paying()">
              {{ paying() ? 'Procesando...' : 'Confirmar pago' }}
            </button>
          </div>
        </div>
      }

      @if (pdfUrl()) {
        <div class="ok">
          ✓ <a [href]="pdfUrl()!" target="_blank" rel="noopener">Abrir PDF generado</a>
        </div>
      }
    }
  `,
  styles: [`
    .back { color: #2c5282; text-decoration: none; font-size: 14px; }
    .back:hover { text-decoration: underline; }
    .card-info { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin: 12px 0 20px; }
    .title { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
    h2 { margin: 0; color: #2d3748; }
    h3 { color: #2d3748; margin: 16px 0 8px; }
    h4 { margin: 0 0 12px; color: #2d3748; font-size: 15px; }
    .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 8px 24px; font-size: 14px; margin-bottom: 12px; }
    code { background: #edf2f7; padding: 2px 6px; border-radius: 4px; font-size: 12px; }
    .actions { display: flex; gap: 8px; flex-wrap: wrap; }
    .btn { padding: 8px 16px; border: 1px solid #cbd5e0; background: white; border-radius: 6px; cursor: pointer; font-size: 14px; }
    .btn.primary { background: #1a365d; color: white; border-color: #1a365d; }
    .btn.success { background: #2f855a; color: white; border-color: #2f855a; }
    .btn.danger { background: #c53030; color: white; border-color: #c53030; }
    .btn:disabled { opacity: 0.6; cursor: not-allowed; }
    .inline-form { background: #f7fafc; padding: 16px; border-radius: 8px; margin-top: 12px; border: 1px solid #e2e8f0; }
    .inline-form.sticky { margin-top: 16px; }
    .inline-form .row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 8px; }
    .inline-form label { display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: #4a5568; }
    .inline-form label.full { display: block; margin-bottom: 8px; }
    .inline-form input, .inline-form select { padding: 8px 10px; border: 1px solid #cbd5e0; border-radius: 6px; font-size: 14px; }
    .inline-form .actions { justify-content: flex-end; margin-top: 12px; }
    .table-wrap { background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); overflow: hidden; }
    table { width: 100%; border-collapse: collapse; font-size: 14px; }
    th, td { padding: 8px 12px; text-align: left; }
    th { background: #f7fafc; color: #4a5568; font-weight: 600; border-bottom: 1px solid #e2e8f0; }
    td { border-bottom: 1px solid #edf2f7; }
    tr.pagada { opacity: 0.65; }
    .r { text-align: right; }
    .badge { padding: 2px 10px; border-radius: 12px; font-size: 12px; }
    .b-activo, .c-pagada { background: #c6f6d5; color: #22543d; }
    .b-pendiente, .c-pendiente { background: #feebc8; color: #7b341e; }
    .b-aprobado { background: #bee3f8; color: #2a4365; }
    .b-finalizado { background: #e2e8f0; color: #4a5568; }
    .b-mora, .b-rechazado, .c-vencida { background: #fed7d7; color: #742a2a; }
    .c-parcial { background: #bee3f8; color: #2a4365; }
    .muted { color: #718096; }
    .center { text-align: center; }
    .totals { display: flex; justify-content: flex-end; gap: 24px; margin-top: 12px; font-size: 14px; color: #4a5568; }
    .hint { color: #718096; font-size: 13px; }
    .err { color: #c53030; background: #fff5f5; padding: 10px; border-radius: 6px; margin: 8px 0; }
    .ok { background: #f0fff4; border: 1px solid #9ae6b4; color: #22543d; padding: 10px; border-radius: 6px; margin-top: 12px; }
    .ok a { color: #22543d; font-weight: 500; }
    .link-btn { background: none; border: none; color: #2c5282; cursor: pointer; font-size: 13px; padding: 0; font-weight: 500; }
    .link-btn:hover { text-decoration: underline; }
  `],
})
export class PrestamoDetail implements OnInit {
  private route = inject(ActivatedRoute);
  private loanSvc = inject(LoanService);
  private paySvc = inject(PaymentService);
  private docSvc = inject(DocumentService);
  private keycloak = inject(KeycloakService);

  prestamo = signal<Prestamo | null>(null);
  cuotas = signal<Cuota[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  generando = signal(false);
  pdfUrl = signal<string | null>(null);

  // Approve
  showApprove = signal(false);
  approving = signal(false);
  approveError = signal<string | null>(null);
  approveForm: { monto_aprobado: number | null; fecha_desembolso: string } = {
    monto_aprobado: null, fecha_desembolso: new Date().toISOString().slice(0, 10),
  };

  // Reject
  showReject = signal(false);
  rejecting = signal(false);
  rejectError = signal<string | null>(null);
  rejectForm = { observaciones: '' };

  // Pay cuota
  payCuota = signal<Cuota | null>(null);
  paying = signal(false);
  payError = signal<string | null>(null);
  payForm: { monto_pagado: number | null; metodo_pago: MetodoPago; observaciones: string } = {
    monto_pagado: null, metodo_pago: 'efectivo', observaciones: '',
  };

  totalCapital = computed(() => this.cuotas().reduce((s, c) => s + c.capital, 0));
  totalInteres = computed(() => this.cuotas().reduce((s, c) => s + c.interes, 0));
  totalGeneral = computed(() => this.cuotas().reduce((s, c) => s + c.total, 0));

  ngOnInit() {
    this.reload();
  }

  reload() {
    const id = this.route.snapshot.paramMap.get('id')!;
    this.loading.set(true);
    this.loanSvc.get(id).subscribe({
      next: p => {
        this.prestamo.set(p);
        this.loanSvc.schedule(id).subscribe({
          next: r => { this.cuotas.set(r.cuotas); this.loading.set(false); },
          error: () => this.loading.set(false),
        });
      },
      error: e => { this.error.set(e.error?.error || e.message); this.loading.set(false); },
    });
  }

  // ─── Aprobar ───
  openApprove() {
    this.showApprove.set(true);
    this.showReject.set(false);
    this.approveError.set(null);
    this.approveForm.monto_aprobado = this.prestamo()?.monto_solicitado ?? null;
  }

  doApprove() {
    const id = this.prestamo()?.id;
    if (!id) return;
    this.approving.set(true);
    this.approveError.set(null);
    this.loanSvc.approve(id, {
      monto_aprobado: this.approveForm.monto_aprobado || undefined,
      fecha_desembolso: this.approveForm.fecha_desembolso || undefined,
      aprobado_por: this.keycloak.userId(),
    }).subscribe({
      next: () => {
        this.approving.set(false);
        this.showApprove.set(false);
        this.reload();
      },
      error: e => {
        this.approveError.set(e.error?.error || e.message);
        this.approving.set(false);
      },
    });
  }

  // ─── Rechazar ───
  openReject() {
    this.showReject.set(true);
    this.showApprove.set(false);
    this.rejectError.set(null);
    this.rejectForm = { observaciones: '' };
  }

  doReject() {
    const id = this.prestamo()?.id;
    if (!id) return;
    this.rejecting.set(true);
    this.rejectError.set(null);
    this.loanSvc.reject(id, {
      aprobado_por: this.keycloak.userId(),
      observaciones: this.rejectForm.observaciones,
    }).subscribe({
      next: () => {
        this.rejecting.set(false);
        this.showReject.set(false);
        this.reload();
      },
      error: e => {
        this.rejectError.set(e.error?.error || e.message);
        this.rejecting.set(false);
      },
    });
  }

  // ─── Pagar cuota ───
  openPay(c: Cuota) {
    this.payCuota.set(c);
    this.payError.set(null);
    this.payForm = {
      monto_pagado: c.saldo_pendiente + c.mora_acumulada,
      metodo_pago: 'efectivo',
      observaciones: '',
    };
  }

  doPay() {
    const c = this.payCuota();
    if (!c || this.payForm.monto_pagado == null) return;
    this.paying.set(true);
    this.payError.set(null);
    this.paySvc.create({
      cuota_id: c.id,
      monto_pagado: this.payForm.monto_pagado,
      metodo_pago: this.payForm.metodo_pago,
      usuario_id: this.keycloak.userId(),
      observaciones: this.payForm.observaciones || undefined,
    }).subscribe({
      next: () => {
        this.paying.set(false);
        this.payCuota.set(null);
        this.reload();
      },
      error: e => {
        this.payError.set(e.error?.error || e.message);
        this.paying.set(false);
      },
    });
  }

  generarContrato() {
    const p = this.prestamo();
    if (!p) return;
    this.generando.set(true);
    this.docSvc.generateContract(p.id).subscribe({
      next: doc => {
        this.pdfUrl.set(this.docSvc.downloadUrl(doc.id));
        this.generando.set(false);
      },
      error: e => {
        this.error.set(e.error?.error || e.message);
        this.generando.set(false);
      },
    });
  }
}
