import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';

import { LoanService, Prestamo, Cuota, Garantia } from '../../core/services/loan.service';
import { PaymentService, MetodoPago } from '../../core/services/payment.service';
import { DocumentService } from '../../core/services/document.service';
import { KeycloakService } from '../../core/keycloak.service';

@Component({
  selector: 'app-prestamo-detail',
  imports: [CommonModule, CurrencyPipe, RouterLink, FormsModule],
  template: `
    <a routerLink="/prestamos" class="text-sm text-navy-light hover:underline">← volver</a>

    @if (loading()) { <p class="mt-3 text-sm text-muted">Cargando...</p> }
    @if (error()) { <p class="mt-3 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ error() }}</p> }

    @if (prestamo(); as p) {
      <div class="mb-5 mt-3 rounded-lg bg-white p-5 shadow-sm">
        <div class="mb-3 flex items-center gap-3">
          <h2 class="m-0 text-xl font-semibold text-ink">Préstamo</h2>
          <span class="rounded-full px-2.5 py-0.5 text-xs font-medium capitalize" [class]="badge(p.estado)">{{ p.estado }}</span>
        </div>
        <div class="mb-3 grid grid-cols-1 gap-x-6 gap-y-2 text-sm text-ink sm:grid-cols-2">
          <div><b>ID:</b> <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">{{ p.id }}</code></div>
          <div><b>Cliente:</b> <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">{{ p.cliente_id }}</code></div>
          <div><b>Monto solicitado:</b> {{ p.monto_solicitado | currency:'BOB':'symbol-narrow':'1.2-2' }}</div>
          <div><b>Monto aprobado:</b> {{ (p.monto_aprobado ?? 0) | currency:'BOB':'symbol-narrow':'1.2-2' }}</div>
          <div><b>Tasa:</b> {{ (p.tasa_interes * 100).toFixed(2) }}% ({{ p.tipo_interes }})</div>
          <div><b>Cuotas:</b> {{ p.num_cuotas }} ({{ p.frecuencia }})</div>
          <div><b>Solicitud:</b> {{ p.fecha_solicitud | slice:0:10 }}</div>
          <div><b>Desembolso:</b> {{ (p.fecha_desembolso | slice:0:10) || '—' }}</div>
          <div><b>Garantía:</b> <span class="capitalize">{{ p.tipo_garantia || 'sin garantía' }}</span></div>
        </div>
        @if (p.observaciones) {
          <p class="text-sm text-ink"><b>Observaciones:</b> {{ p.observaciones }}</p>
        }

        <!-- Acciones disponibles según estado -->
        <div class="flex flex-wrap gap-2">
          @if (p.estado === 'pendiente') {
            <button (click)="openApprove()"
                    class="rounded-md bg-green-700 px-4 py-2 text-sm font-medium text-white transition hover:bg-green-800">✓ Aprobar</button>
            <button (click)="openReject()"
                    class="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-red-700">✗ Rechazar</button>
          }
          <button (click)="generarContrato()" [disabled]="generando()"
                  class="rounded-md bg-navy px-4 py-2 text-sm font-medium text-white transition hover:bg-navy-light disabled:cursor-not-allowed disabled:opacity-50">
            {{ generando() ? 'Generando...' : '📄 Contrato PDF' }}
          </button>
        </div>

        <!-- Form Aprobar -->
        @if (showApprove()) {
          <div class="mt-3 rounded-lg border border-slate-200 bg-slate-50 p-4">
            <h4 class="mb-3 text-sm font-semibold text-ink">Aprobar préstamo</h4>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <label class="flex flex-col gap-1 text-sm text-slate-600">Monto aprobado (default = solicitado)
                <input type="number" min="1" step="0.01"
                       [(ngModel)]="approveForm.monto_aprobado" name="monto_a" class="ui-input">
              </label>
              <label class="flex flex-col gap-1 text-sm text-slate-600">Fecha desembolso
                <input type="date" [(ngModel)]="approveForm.fecha_desembolso" name="fecha_d" class="ui-input">
              </label>
            </div>
            @if (approveError()) { <p class="mt-2 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ approveError() }}</p> }
            <div class="mt-3 flex flex-wrap justify-end gap-2">
              <button (click)="showApprove.set(false)"
                      class="rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-50">Cancelar</button>
              <button (click)="doApprove()" [disabled]="approving()"
                      class="rounded-md bg-green-700 px-4 py-2 text-sm font-medium text-white transition hover:bg-green-800 disabled:cursor-not-allowed disabled:opacity-50">
                {{ approving() ? 'Aprobando...' : 'Confirmar aprobación' }}
              </button>
            </div>
          </div>
        }

        <!-- Form Rechazar -->
        @if (showReject()) {
          <div class="mt-3 rounded-lg border border-slate-200 bg-slate-50 p-4">
            <h4 class="mb-3 text-sm font-semibold text-ink">Rechazar préstamo</h4>
            <label class="flex flex-col gap-1 text-sm text-slate-600">Motivo *
              <input [(ngModel)]="rejectForm.observaciones" name="obs" required minlength="3" class="ui-input">
            </label>
            @if (rejectError()) { <p class="mt-2 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ rejectError() }}</p> }
            <div class="mt-3 flex flex-wrap justify-end gap-2">
              <button (click)="showReject.set(false)"
                      class="rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-50">Cancelar</button>
              <button (click)="doReject()" [disabled]="rejecting() || !rejectForm.observaciones"
                      class="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50">
                {{ rejecting() ? 'Rechazando...' : 'Confirmar rechazo' }}
              </button>
            </div>
          </div>
        }
      </div>

      <!-- Garantía: tipo + imágenes adjuntas -->
      <div class="mb-5 rounded-lg bg-white p-5 shadow-sm">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h3 class="m-0 text-base font-semibold text-ink">
            Garantía
            @if (p.tipo_garantia) {
              <span class="ml-2 rounded-full bg-slate-200 px-2.5 py-0.5 text-xs font-medium capitalize text-slate-700">{{ p.tipo_garantia }}</span>
            } @else {
              <span class="ml-2 text-sm font-normal text-muted">sin garantía</span>
            }
          </h3>
          @if (puedeEditar()) {
            <label class="cursor-pointer rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-50">
              + Imagen
              <input type="file" accept="image/png,image/jpeg,image/webp" class="hidden" (change)="onFile($event)">
            </label>
          }
        </div>

        @if (selectedName()) {
          <div class="mb-3 flex flex-wrap items-center gap-2 rounded-md bg-slate-50 p-2 text-sm">
            <span class="text-ink">{{ selectedName() }}</span>
            <button (click)="doUpload()" [disabled]="uploading()"
                    class="rounded-md bg-navy px-3 py-1 text-xs font-medium text-white hover:bg-navy-light disabled:opacity-50">
              {{ uploading() ? 'Subiendo...' : 'Subir' }}
            </button>
            <button (click)="clearFile()" class="text-xs text-muted hover:underline">cancelar</button>
          </div>
        }
        @if (uploadError()) { <p class="mb-3 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ uploadError() }}</p> }

        @if (garantias().length > 0) {
          <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
            @for (g of garantias(); track g.id) {
              <figure class="m-0 overflow-hidden rounded-lg border border-slate-200">
                <a [href]="thumbs()[g.id]" target="_blank" rel="noopener">
                  <img [src]="thumbs()[g.id]" [alt]="g.nombre_archivo" class="h-32 w-full bg-slate-100 object-cover"/>
                </a>
                <figcaption class="flex items-center justify-between gap-2 px-2 py-1 text-xs text-muted">
                  <span class="truncate" [title]="g.nombre_archivo">{{ g.nombre_archivo }}</span>
                  @if (puedeEditar()) {
                    <button (click)="removeGarantia(g.id)" class="shrink-0 text-red-600 hover:underline" title="Eliminar">✕</button>
                  }
                </figcaption>
              </figure>
            }
          </div>
        } @else {
          <p class="text-sm text-muted">Sin imágenes adjuntas.</p>
        }
      </div>

      <h3 class="mb-3 text-base font-semibold text-ink">Plan de pagos</h3>
      <div class="overflow-x-auto rounded-lg bg-white shadow-sm">
        <table class="w-full min-w-[820px] border-collapse text-sm">
          <thead>
            <tr class="border-b border-slate-200 bg-slate-50 text-left text-slate-600">
              <th class="px-3 py-2 text-center font-semibold">#</th>
              <th class="px-3 py-2 font-semibold">Vencimiento</th>
              <th class="px-3 py-2 text-right font-semibold">Capital</th>
              <th class="px-3 py-2 text-right font-semibold">Interés</th>
              <th class="px-3 py-2 text-right font-semibold">Total</th>
              <th class="px-3 py-2 text-right font-semibold">Saldo</th>
              <th class="px-3 py-2 font-semibold">Estado</th>
              <th class="px-3 py-2 font-semibold">Pago</th>
              <th class="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            @for (c of cuotas(); track c.id) {
              <tr class="border-b border-slate-100 last:border-0" [class.opacity-55]="c.estado === 'pagada'">
                <td class="px-3 py-2 text-center">{{ c.numero }}</td>
                <td class="px-3 py-2">{{ c.fecha_vencimiento | slice:0:10 }}</td>
                <td class="px-3 py-2 text-right">{{ c.capital | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td class="px-3 py-2 text-right">{{ c.interes | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td class="px-3 py-2 text-right font-semibold">{{ c.total | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td class="px-3 py-2 text-right">{{ c.saldo_pendiente | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
                <td class="px-3 py-2"><span class="rounded-full px-2.5 py-0.5 text-xs font-medium capitalize" [class]="badge(c.estado)">{{ c.estado }}</span></td>
                <td class="px-3 py-2 text-muted">{{ (c.fecha_pago | slice:0:10) || '—' }}</td>
                <td class="px-3 py-2">
                  @if (c.estado !== 'pagada' && p.estado === 'activo') {
                    <button (click)="openPay(c)" class="font-semibold text-navy-light hover:underline">💰 Pagar</button>
                  }
                </td>
              </tr>
            } @empty {
              <tr><td colspan="9" class="px-3 py-6 text-center text-muted">Sin cuotas (préstamo aún no aprobado)</td></tr>
            }
          </tbody>
        </table>
      </div>

      @if (cuotas().length > 0) {
        <div class="mt-3 flex flex-wrap justify-end gap-6 text-sm text-slate-600">
          <span>Capital: <b class="text-ink">{{ totalCapital() | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></span>
          <span>Interés: <b class="text-ink">{{ totalInteres() | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></span>
          <span>Total: <b class="text-ink">{{ totalGeneral() | currency:'BOB':'symbol-narrow':'1.2-2' }}</b></span>
        </div>
      }

      <!-- Form Pagar Cuota -->
      @if (payCuota(); as pc) {
        <div class="mt-4 rounded-lg border border-slate-200 bg-slate-50 p-4">
          <h4 class="mb-3 text-sm font-semibold text-ink">Registrar pago — Cuota #{{ pc.numero }}</h4>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label class="flex flex-col gap-1 text-sm text-slate-600">Monto a pagar *
              <input type="number" min="0.01" step="0.01" max="{{ pc.saldo_pendiente + pc.mora_acumulada }}"
                     [(ngModel)]="payForm.monto_pagado" name="monto_p" required class="ui-input">
            </label>
            <label class="flex flex-col gap-1 text-sm text-slate-600">Método de pago *
              <select [(ngModel)]="payForm.metodo_pago" name="metodo_p" required class="ui-input">
                <option value="efectivo">Efectivo</option>
                <option value="transferencia">Transferencia</option>
                <option value="cheque">Cheque</option>
                <option value="tarjeta">Tarjeta</option>
                <option value="qr">QR</option>
              </select>
            </label>
          </div>
          <label class="mt-3 flex flex-col gap-1 text-sm text-slate-600">Observaciones
            <input [(ngModel)]="payForm.observaciones" name="obs_p" class="ui-input">
          </label>
          <p class="mt-2 text-sm text-muted">
            Saldo cuota: <b class="text-ink">{{ pc.saldo_pendiente | currency:'BOB':'symbol-narrow':'1.2-2' }}</b>
            @if (pc.mora_acumulada > 0) {
              · Mora: <b class="text-red-600">{{ pc.mora_acumulada | currency:'BOB':'symbol-narrow':'1.2-2' }}</b>
            }
          </p>
          @if (payError()) { <p class="mt-2 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ payError() }}</p> }
          <div class="mt-3 flex flex-wrap justify-end gap-2">
            <button (click)="payCuota.set(null)"
                    class="rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-50">Cancelar</button>
            <button (click)="doPay()" [disabled]="paying()"
                    class="rounded-md bg-navy px-4 py-2 text-sm font-medium text-white transition hover:bg-navy-light disabled:cursor-not-allowed disabled:opacity-50">
              {{ paying() ? 'Procesando...' : 'Confirmar pago' }}
            </button>
          </div>
        </div>
      }

      @if (pdfUrl()) {
        <div class="mt-3 rounded-md border border-green-200 bg-green-50 p-3 text-sm text-green-800">
          ✓ <a [href]="pdfUrl()!" target="_blank" rel="noopener" class="font-semibold hover:underline">Abrir PDF generado</a>
        </div>
      }
    }
  `,
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
  // Clave de idempotencia del intento de pago en curso (estable entre reintentos).
  private payKey: string | null = null;

  // Garantías (imágenes)
  garantias = signal<Garantia[]>([]);
  thumbs = signal<Record<string, string>>({}); // gid -> object URL
  uploading = signal(false);
  uploadError = signal<string | null>(null);
  selectedName = signal<string | null>(null);
  private selectedFile: File | null = null;
  // Subir/eliminar garantías: cajero/admin (igual que el backend).
  puedeEditar = computed(() =>
    this.keycloak.roles().includes('admin') || this.keycloak.roles().includes('cajero'),
  );

  // Color de badge por estado de préstamo o de cuota.
  badge(estado: string): string {
    switch (estado) {
      case 'activo':
      case 'pagada':
        return 'bg-green-100 text-green-800';
      case 'mora':
      case 'rechazado':
      case 'vencida':
        return 'bg-red-100 text-red-800';
      case 'finalizado':
        return 'bg-slate-200 text-slate-700';
      case 'pendiente':
      case 'aprobado':
      case 'parcial':
        return 'bg-orange-100 text-orange-800';
      default:
        return 'bg-slate-200 text-slate-600';
    }
  }

  totalCapital = computed(() => this.cuotas().reduce((s, c) => s + c.capital, 0));
  totalInteres = computed(() => this.cuotas().reduce((s, c) => s + c.interes, 0));
  totalGeneral = computed(() => this.cuotas().reduce((s, c) => s + c.total, 0));

  ngOnInit() {
    this.reload();
    this.loadGarantias();
  }

  // ─── Garantías ───
  loadGarantias() {
    const id = this.route.snapshot.paramMap.get('id');
    if (!id) return;
    this.loanSvc.listGarantias(id).subscribe({
      next: r => {
        this.garantias.set(r.items);
        // Revoca miniaturas anteriores y descarga las nuevas (autenticado → blob).
        Object.values(this.thumbs()).forEach(u => URL.revokeObjectURL(u));
        this.thumbs.set({});
        for (const g of r.items) {
          this.loanSvc.downloadGarantia(id, g.id).subscribe({
            next: blob => this.thumbs.update(t => ({ ...t, [g.id]: URL.createObjectURL(blob) })),
          });
        }
      },
    });
  }

  onFile(ev: Event) {
    const input = ev.target as HTMLInputElement;
    this.selectedFile = input.files?.[0] ?? null;
    this.selectedName.set(this.selectedFile?.name ?? null);
    this.uploadError.set(null);
  }

  clearFile() {
    this.selectedFile = null;
    this.selectedName.set(null);
  }

  doUpload() {
    const id = this.prestamo()?.id;
    if (!id || !this.selectedFile) return;
    this.uploading.set(true);
    this.uploadError.set(null);
    this.loanSvc.uploadGarantia(id, this.selectedFile).subscribe({
      next: () => { this.uploading.set(false); this.clearFile(); this.loadGarantias(); },
      error: e => { this.uploadError.set(e.error?.error || e.message); this.uploading.set(false); },
    });
  }

  removeGarantia(gid: string) {
    const id = this.prestamo()?.id;
    if (!id) return;
    this.loanSvc.deleteGarantia(id, gid).subscribe({ next: () => this.loadGarantias() });
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
    // Una clave por intento de pago: reintentos reusan la misma → sin doble cobro.
    this.payKey = crypto.randomUUID();
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
    }, this.payKey ?? undefined).subscribe({
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
        // Descarga autenticada → blob URL (el gateway exige JWT; un href directo
        // al endpoint no llevaría el Bearer token y daría 401).
        this.docSvc.download(doc.id).subscribe({
          next: blob => { this.pdfUrl.set(URL.createObjectURL(blob)); this.generando.set(false); },
          error: e => { this.error.set(e.error?.error || e.message); this.generando.set(false); },
        });
      },
      error: e => {
        this.error.set(e.error?.error || e.message);
        this.generando.set(false);
      },
    });
  }
}
