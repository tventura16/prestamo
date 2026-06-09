import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { PaymentService, Pago } from '../../core/services/payment.service';
import { DocumentService } from '../../core/services/document.service';
import { KeycloakService } from '../../core/keycloak.service';

@Component({
  selector: 'app-pagos',
  imports: [CommonModule, CurrencyPipe],
  template: `
    <h2 class="text-xl font-semibold text-ink">Pagos</h2>

    @if (loading()) { <p class="mt-3 text-muted">Cargando...</p> }
    @if (error()) { <p class="mt-3 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ error() }}</p> }

    <div class="mt-3 overflow-x-auto rounded-lg bg-white shadow-sm">
      <table class="w-full min-w-[900px] border-collapse text-sm">
        <thead>
          <tr class="border-b border-slate-200 bg-slate-50 text-left text-slate-600">
            <th class="px-3 py-2 font-semibold">Recibo</th>
            <th class="px-3 py-2 font-semibold">Fecha</th>
            <th class="px-3 py-2 font-semibold">Cliente</th>
            <th class="px-3 py-2 font-semibold">Préstamo</th>
            <th class="px-3 py-2 text-right font-semibold">Monto</th>
            <th class="px-3 py-2 text-right font-semibold">Capital</th>
            <th class="px-3 py-2 text-right font-semibold">Interés</th>
            <th class="px-3 py-2 text-right font-semibold">Mora</th>
            <th class="px-3 py-2 font-semibold">Método</th>
            <th class="px-3 py-2 font-semibold">Tipo</th>
            <th class="px-3 py-2"></th>
          </tr>
        </thead>
        <tbody>
          @for (p of items(); track p.id) {
            <tr class="border-b border-slate-100 last:border-0" [class.opacity-55]="p.anulado">
              <td class="px-3 py-2 font-semibold" [class.line-through]="p.anulado">{{ p.numero_recibo }}</td>
              <td class="px-3 py-2 text-muted" [class.line-through]="p.anulado">{{ p.fecha_pago | slice:0:19 | slice:0:10 }} {{ p.fecha_pago | slice:11:16 }}</td>
              <td class="px-3 py-2" [class.line-through]="p.anulado"><code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">{{ p.cliente_id | slice:0:8 }}</code></td>
              <td class="px-3 py-2" [class.line-through]="p.anulado"><code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">{{ p.prestamo_id | slice:0:8 }}</code></td>
              <td class="px-3 py-2 text-right font-semibold" [class.line-through]="p.anulado">{{ p.monto_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
              <td class="px-3 py-2 text-right" [class.line-through]="p.anulado">{{ p.capital_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
              <td class="px-3 py-2 text-right" [class.line-through]="p.anulado">{{ p.interes_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
              <td class="px-3 py-2 text-right" [class.line-through]="p.anulado">{{ p.mora_pagada | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
              <td class="px-3 py-2 capitalize" [class.line-through]="p.anulado">{{ p.metodo_pago }}</td>
              <td class="px-3 py-2" [class.line-through]="p.anulado"><span class="rounded-full px-2.5 py-0.5 text-xs font-medium capitalize" [class]="badge(p.tipo)">{{ p.tipo }}</span></td>
              <td class="px-3 py-2">
                <div class="flex items-center gap-3 whitespace-nowrap">
                  <button class="text-navy-light hover:underline disabled:opacity-50" (click)="generarRecibo(p)" [disabled]="generando() === p.id">
                    {{ generando() === p.id ? '...' : 'Recibo PDF' }}
                  </button>
                  @if (p.anulado) {
                    <span class="rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-800" [title]="p.motivo_anulacion || ''">Anulado</span>
                  } @else if (puedeAnular()) {
                    <button class="text-red-600 hover:underline" (click)="abrirAnular(p)">Anular</button>
                  }
                </div>
              </td>
            </tr>
          } @empty {
            <tr><td colspan="11" class="px-3 py-6 text-center text-muted">Sin pagos registrados</td></tr>
          }
        </tbody>
      </table>
    </div>

    <p class="mt-2 text-sm text-muted">Mostrando {{ items().length }} de {{ total() }} pagos</p>

    @if (anulandoPago(); as p) {
      <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" (click)="cerrarAnular()">
        <div class="w-full max-w-md rounded-xl bg-white p-6 shadow-2xl" (click)="$event.stopPropagation()" role="dialog" aria-modal="true" aria-labelledby="anular-title">
          <h3 id="anular-title" class="text-lg font-semibold text-ink">Anular pago {{ p.numero_recibo }}</h3>
          <p class="mt-2 text-sm text-muted">
            Se revertirá la aplicación a la cuota
            (<b>{{ p.monto_pagado | currency:'BOB':'symbol-narrow':'1.2-2' }}</b>) y la acción quedará
            registrada en auditoría. No se puede deshacer.
          </p>
          <label for="motivo" class="mt-4 mb-1 block text-sm font-medium text-ink">Motivo <span class="text-red-600">*</span></label>
          <textarea id="motivo" rows="3" class="ui-input w-full" [value]="motivo()"
                    (input)="motivo.set($any($event.target).value)"
                    placeholder="Ej.: pago duplicado por error de caja"></textarea>
          @if (modalError()) { <p class="mt-3 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ modalError() }}</p> }
          <div class="mt-4 flex justify-end gap-2">
            <button class="rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-50 disabled:opacity-50" (click)="cerrarAnular()" [disabled]="anulando()">Cancelar</button>
            <button class="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50" (click)="confirmarAnular()" [disabled]="anulando() || !motivo().trim()">
              {{ anulando() ? 'Anulando...' : 'Anular pago' }}
            </button>
          </div>
        </div>
      </div>
    }
  `,
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

  // Color de badge por tipo de pago (mismo vocabulario que cliente-detail).
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
