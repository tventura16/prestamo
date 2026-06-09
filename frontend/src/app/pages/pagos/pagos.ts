import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { PaymentService, Pago } from '../../core/services/payment.service';
import { DocumentService } from '../../core/services/document.service';
import { KeycloakService } from '../../core/keycloak.service';

@Component({
  selector: 'app-pagos',
  imports: [CommonModule, CurrencyPipe],
  template: `
    <h2>Pagos</h2>

    @if (loading()) { <p class="hint">Cargando...</p> }
    @if (error()) { <p class="err">{{ error() }}</p> }

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Recibo</th>
            <th>Fecha</th>
            <th>Cliente</th>
            <th>Préstamo</th>
            <th class="r">Monto</th>
            <th class="r">Capital</th>
            <th class="r">Interés</th>
            <th class="r">Mora</th>
            <th>Método</th>
            <th>Tipo</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          @for (p of items(); track p.id) {
            <tr [class.anulado]="p.anulado">
              <td><b>{{ p.numero_recibo }}</b></td>
              <td class="muted">{{ p.fecha_pago | slice:0:19 | slice:0:10 }} {{ p.fecha_pago | slice:11:16 }}</td>
              <td><code>{{ p.cliente_id | slice:0:8 }}</code></td>
              <td><code>{{ p.prestamo_id | slice:0:8 }}</code></td>
              <td class="r"><b>{{ p.monto_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></td>
              <td class="r">{{ p.capital_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
              <td class="r">{{ p.interes_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
              <td class="r">{{ p.mora_pagada | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
              <td>{{ p.metodo_pago }}</td>
              <td><span class="badge" [class]="'t-' + p.tipo">{{ p.tipo }}</span></td>
              <td class="actions">
                <button class="link-btn" (click)="generarRecibo(p)" [disabled]="generando() === p.id">
                  {{ generando() === p.id ? '...' : 'Recibo PDF' }}
                </button>
                @if (p.anulado) {
                  <span class="badge anulado-badge" [title]="p.motivo_anulacion || ''">Anulado</span>
                } @else if (puedeAnular()) {
                  <button class="link-btn danger" (click)="abrirAnular(p)">Anular</button>
                }
              </td>
            </tr>
          } @empty {
            <tr><td colspan="11" class="muted center">Sin pagos registrados</td></tr>
          }
        </tbody>
      </table>
    </div>

    <p class="hint">Mostrando {{ items().length }} de {{ total() }} pagos</p>

    @if (anulandoPago(); as p) {
      <div class="modal-overlay" (click)="cerrarAnular()">
        <div class="modal" (click)="$event.stopPropagation()" role="dialog" aria-modal="true" aria-labelledby="anular-title">
          <h3 id="anular-title">Anular pago {{ p.numero_recibo }}</h3>
          <p class="muted">
            Se revertirá la aplicación a la cuota
            (<b>{{ p.monto_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</b>) y la acción quedará
            registrada en auditoría. No se puede deshacer.
          </p>
          <label for="motivo">Motivo <span class="req">*</span></label>
          <textarea id="motivo" rows="3" [value]="motivo()"
                    (input)="motivo.set($any($event.target).value)"
                    placeholder="Ej.: pago duplicado por error de caja"></textarea>
          @if (modalError()) { <p class="err">{{ modalError() }}</p> }
          <div class="modal-actions">
            <button class="btn-ghost" (click)="cerrarAnular()" [disabled]="anulando()">Cancelar</button>
            <button class="btn-danger" (click)="confirmarAnular()" [disabled]="anulando() || !motivo().trim()">
              {{ anulando() ? 'Anulando...' : 'Anular pago' }}
            </button>
          </div>
        </div>
      </div>
    }
  `,
  styles: [`
    h2 { color: #2d3748; }
    .table-wrap { background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); overflow: hidden; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    th, td { padding: 8px 12px; text-align: left; }
    th { background: #f7fafc; color: #4a5568; font-weight: 600; border-bottom: 1px solid #e2e8f0; }
    td { border-bottom: 1px solid #edf2f7; }
    .r { text-align: right; }
    code { background: #edf2f7; padding: 2px 6px; border-radius: 4px; font-size: 11px; }
    .badge { padding: 2px 10px; border-radius: 12px; font-size: 12px; }
    .t-total { background: #c6f6d5; color: #22543d; }
    .t-parcial { background: #bee3f8; color: #2a4365; }
    .muted { color: #718096; }
    .center { text-align: center; }
    .hint { color: #718096; font-size: 13px; margin-top: 8px; }
    .err { color: #c53030; background: #fff5f5; padding: 10px; border-radius: 6px; }
    tr.anulado { opacity: 0.55; }
    tr.anulado td:not(.actions) { text-decoration: line-through; }
    .actions { display: flex; gap: 12px; align-items: center; white-space: nowrap; }
    .link-btn { background: none; border: none; color: #2c5282; cursor: pointer; font-size: 13px; padding: 0; }
    .link-btn:hover { text-decoration: underline; }
    .link-btn:disabled { color: #a0aec0; cursor: not-allowed; }
    .link-btn.danger { color: #c53030; }
    .anulado-badge { background: #fed7d7; color: #822727; cursor: default; }
    .modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.45); display: flex;
      align-items: center; justify-content: center; z-index: 50; }
    .modal { background: white; border-radius: 10px; padding: 24px; width: 100%; max-width: 440px;
      box-shadow: 0 10px 40px rgba(0,0,0,0.25); }
    .modal h3 { margin: 0 0 8px; color: #2d3748; }
    .modal label { display: block; font-size: 13px; font-weight: 600; color: #4a5568; margin: 12px 0 4px; }
    .modal .req { color: #c53030; }
    .modal textarea { width: 100%; box-sizing: border-box; border: 1px solid #cbd5e0; border-radius: 6px;
      padding: 8px; font: inherit; font-size: 14px; resize: vertical; }
    .modal textarea:focus { outline: none; border-color: #3182ce; box-shadow: 0 0 0 3px rgba(49,130,206,0.15); }
    .modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; }
    .btn-ghost { background: none; border: 1px solid #cbd5e0; color: #4a5568; border-radius: 6px;
      padding: 8px 16px; cursor: pointer; font-size: 14px; }
    .btn-danger { background: #c53030; border: none; color: white; border-radius: 6px;
      padding: 8px 16px; cursor: pointer; font-size: 14px; }
    .btn-danger:disabled, .btn-ghost:disabled { opacity: 0.5; cursor: not-allowed; }
  `],
})
export class Pagos implements OnInit {
  private svc = inject(PaymentService);
  private docSvc = inject(DocumentService);
  private kc = inject(KeycloakService);

  items = signal<Pago[]>([]);
  total = signal(0);
  loading = signal(false);
  error = signal<string | null>(null);
  generando = signal<string | null>(null);

  // Anular es operación sensible: visible solo para supervisor/admin (el
  // backend lo aplica igual; esto evita ofrecer una acción que daría 403).
  puedeAnular = computed(() =>
    this.kc.roles().includes('admin') || this.kc.roles().includes('supervisor'),
  );

  anulandoPago = signal<Pago | null>(null);
  motivo = signal('');
  anulando = signal(false);
  modalError = signal<string | null>(null);

  ngOnInit() {
    this.loading.set(true);
    this.svc.list({ limit: 100 }).subscribe({
      next: r => { this.items.set(r.items); this.total.set(r.total); this.loading.set(false); },
      error: e => { this.error.set(e.error?.error || e.message); this.loading.set(false); },
    });
  }

  generarRecibo(p: Pago) {
    this.generando.set(p.id);
    this.docSvc.generateReceipt(p.id).subscribe({
      next: doc => {
        // Descarga autenticada (el gateway exige JWT; window.open no lleva token).
        this.docSvc.download(doc.id).subscribe({
          next: blob => {
            this.generando.set(null);
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = doc.nombre_archivo || `recibo-${doc.id}.pdf`;
            a.click();
            URL.revokeObjectURL(url);
          },
          error: e => { this.error.set(e.error?.error || e.message); this.generando.set(null); },
        });
      },
      error: e => {
        this.error.set(e.error?.error || e.message);
        this.generando.set(null);
      },
    });
  }

  abrirAnular(p: Pago) {
    this.anulandoPago.set(p);
    this.motivo.set('');
    this.modalError.set(null);
  }

  cerrarAnular() {
    if (this.anulando()) return;
    this.anulandoPago.set(null);
  }

  confirmarAnular() {
    const p = this.anulandoPago();
    const motivo = this.motivo().trim();
    if (!p || !motivo) return;

    this.anulando.set(true);
    this.modalError.set(null);
    this.svc.anular(p.id, motivo).subscribe({
      next: res => {
        // Reemplaza la fila con el pago anulado devuelto por el backend.
        this.items.update(list => list.map(x => (x.id === p.id ? res.pago : x)));
        this.anulando.set(false);
        this.anulandoPago.set(null);
      },
      error: e => {
        this.modalError.set(e.error?.error || e.message);
        this.anulando.set(false);
      },
    });
  }
}
