import { Component, OnInit, inject, signal, computed } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';

import { LoanService, Prestamo, Cuota, Garantia, SubtipoGarantia } from '../../core/services/loan.service';
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

      <!-- Garantías (entidades con datos por subtipo + imágenes) -->
      <div class="mb-5 rounded-lg bg-white p-5 shadow-sm">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h3 class="m-0 text-base font-semibold text-ink">Garantías <span class="text-sm font-normal text-muted">({{ garantias().length }})</span></h3>
          @if (puedeEditar()) {
            <button (click)="toggleNuevaGarantia()"
                    class="rounded-md border border-slate-300 px-3 py-1.5 text-sm hover:bg-slate-50">
              {{ mostrarNueva() ? 'Cancelar' : '+ Garantía' }}
            </button>
          }
        </div>

        <!-- Form nueva garantía -->
        @if (mostrarNueva()) {
          <div class="mb-4 rounded-lg border border-slate-200 bg-slate-50 p-4">
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <label class="flex flex-col gap-1 text-sm text-slate-600">Tipo de garantía *
                <select [(ngModel)]="nuevoSubtipo" (change)="onSubtipoChange()" name="subtipo" class="ui-input">
                  <option value="vehiculo">Vehículo</option>
                  <option value="inmueble">Inmueble (hipotecaria)</option>
                  <option value="garante">Garante (fianza)</option>
                  <option value="mueble">Bien mueble / mercadería</option>
                </select>
              </label>
              <label class="flex flex-col gap-1 text-sm text-slate-600">Valor estimado (BOB)
                <input type="number" min="0" step="0.01" [(ngModel)]="nuevoValor" name="valor" class="ui-input">
              </label>
            </div>
            <!-- Campos dinámicos según subtipo -->
            <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
              @for (campo of camposActuales(); track campo.key) {
                <label class="flex flex-col gap-1 text-sm text-slate-600">{{ campo.label }}{{ campo.required ? ' *' : '' }}
                  <input [type]="campo.type || 'text'" [(ngModel)]="nuevoDatos[campo.key]" [name]="'d_' + campo.key" class="ui-input">
                </label>
              }
            </div>
            <label class="mt-3 flex flex-col gap-1 text-sm text-slate-600">Descripción / observación
              <input [(ngModel)]="nuevoDescripcion" name="g_desc" class="ui-input">
            </label>
            @if (nuevaError()) { <p class="mt-2 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ nuevaError() }}</p> }
            <div class="mt-3 flex flex-wrap justify-end gap-2">
              <button (click)="toggleNuevaGarantia()" class="rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-50">Cancelar</button>
              <button (click)="crearGarantia()" [disabled]="guardandoGarantia()"
                      class="rounded-md bg-navy px-4 py-2 text-sm font-medium text-white hover:bg-navy-light disabled:opacity-50">
                {{ guardandoGarantia() ? 'Guardando...' : 'Agregar garantía' }}
              </button>
            </div>
          </div>
        }

        @if (garantias().length === 0 && !mostrarNueva()) {
          <p class="text-sm text-muted">Sin garantías registradas.</p>
        }

        <!-- Lista de garantías -->
        <div class="flex flex-col gap-4">
          @for (g of garantias(); track g.id) {
            <div class="rounded-lg border border-slate-200 p-4">
              <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
                <div class="flex items-center gap-2">
                  <span class="rounded-full bg-navy px-2.5 py-0.5 text-xs font-medium capitalize text-white">{{ subtipoLabel(g.subtipo) }}</span>
                  @if (g.valor_estimado) {
                    <span class="text-sm text-muted">{{ g.valor_estimado | currency:g.moneda:'symbol-narrow':'1.2-2' }}</span>
                  }
                </div>
                <div class="flex items-center gap-2">
                  @if (puedeEditar()) {
                    <label class="cursor-pointer text-xs text-navy-light hover:underline">
                      + imagen
                      <input type="file" accept="image/png,image/jpeg,image/webp" class="hidden" (change)="onFileGarantia($event, g.id)">
                    </label>
                    <button (click)="eliminarGarantia(g.id)" class="text-xs text-red-600 hover:underline">eliminar</button>
                  }
                </div>
              </div>
              <!-- Datos del subtipo -->
              <div class="mb-2 grid grid-cols-1 gap-x-6 gap-y-1 text-sm text-ink sm:grid-cols-2">
                @for (campo of camposDe(g.subtipo); track campo.key) {
                  @if (g.datos[campo.key]) {
                    <div><b>{{ campo.label }}:</b> {{ g.datos[campo.key] }}</div>
                  }
                }
              </div>
              @if (g.descripcion) { <p class="mb-2 text-sm text-muted">{{ g.descripcion }}</p> }
              @if (subiendoEn() === g.id) { <p class="mb-2 text-xs text-muted">Subiendo imagen...</p> }
              <!-- Galería de imágenes de esta garantía -->
              @if (g.imagenes && g.imagenes.length > 0) {
                <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
                  @for (img of g.imagenes; track img.id) {
                    <figure class="m-0 overflow-hidden rounded-lg border border-slate-200">
                      <a [href]="thumbs()[img.id]" target="_blank" rel="noopener">
                        <img [src]="thumbs()[img.id]" [alt]="img.nombre_archivo" class="h-28 w-full bg-slate-100 object-cover"/>
                      </a>
                      <figcaption class="flex items-center justify-between gap-1 px-2 py-1 text-xs text-muted">
                        <span class="truncate" [title]="img.nombre_archivo">{{ img.nombre_archivo }}</span>
                        @if (puedeEditar()) {
                          <button (click)="eliminarImagen(g.id, img.id)" class="shrink-0 text-red-600 hover:underline">✕</button>
                        }
                      </figcaption>
                    </figure>
                  }
                </div>
              } @else {
                <p class="text-xs text-muted">Sin imágenes.</p>
              }
            </div>
          }
        </div>
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

  // ─── Garantías ───
  garantias = signal<Garantia[]>([]);
  thumbs = signal<Record<string, string>>({}); // imagen.id -> object URL
  // Subir/editar garantías: cajero/admin (igual que el backend).
  puedeEditar = computed(() =>
    this.keycloak.roles().includes('admin') || this.keycloak.roles().includes('cajero'),
  );

  // Form de nueva garantía
  mostrarNueva = signal(false);
  guardandoGarantia = signal(false);
  nuevaError = signal<string | null>(null);
  subiendoEn = signal<string | null>(null); // gid en subida de imagen
  nuevoSubtipo: SubtipoGarantia = 'vehiculo';
  nuevoValor: number | null = null;
  nuevoDescripcion = '';
  nuevoDatos: Record<string, any> = {};

  // Campos de cada subtipo (deben coincidir con la validación del backend).
  private readonly camposPorSubtipo: Record<SubtipoGarantia, { key: string; label: string; required?: boolean; type?: string }[]> = {
    vehiculo: [
      { key: 'placa', label: 'Placa', required: true },
      { key: 'marca', label: 'Marca', required: true },
      { key: 'modelo', label: 'Modelo' },
      { key: 'anio', label: 'Año', type: 'number' },
      { key: 'color', label: 'Color' },
      { key: 'nro_motor', label: 'Nº motor' },
      { key: 'nro_chasis', label: 'Nº chasis' },
    ],
    inmueble: [
      { key: 'tipo_inmueble', label: 'Tipo (casa/terreno/local)', required: true },
      { key: 'direccion', label: 'Dirección', required: true },
      { key: 'matricula_folio', label: 'Matrícula / folio' },
      { key: 'superficie_m2', label: 'Superficie (m²)', type: 'number' },
      { key: 'gravamenes', label: 'Gravámenes' },
    ],
    garante: [
      { key: 'nombres', label: 'Nombres', required: true },
      { key: 'apellidos', label: 'Apellidos' },
      { key: 'ci', label: 'CI', required: true },
      { key: 'telefono', label: 'Teléfono' },
      { key: 'direccion', label: 'Dirección' },
      { key: 'actividad', label: 'Actividad' },
    ],
    mueble: [
      { key: 'descripcion', label: 'Descripción', required: true },
      { key: 'ubicacion', label: 'Ubicación' },
      { key: 'marca', label: 'Marca' },
      { key: 'cantidad', label: 'Cantidad', type: 'number' },
    ],
  };

  camposActuales() { return this.camposPorSubtipo[this.nuevoSubtipo]; }
  camposDe(subtipo: SubtipoGarantia) { return this.camposPorSubtipo[subtipo]; }
  subtipoLabel(s: SubtipoGarantia): string {
    return { vehiculo: 'Vehículo', inmueble: 'Inmueble', garante: 'Garante', mueble: 'Bien mueble' }[s] ?? s;
  }

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
        // Revoca miniaturas anteriores y descarga las imágenes (autenticado → blob).
        Object.values(this.thumbs()).forEach(u => URL.revokeObjectURL(u));
        this.thumbs.set({});
        for (const g of r.items) {
          for (const img of g.imagenes ?? []) {
            this.loanSvc.downloadImagen(id, g.id, img.id).subscribe({
              next: blob => this.thumbs.update(t => ({ ...t, [img.id]: URL.createObjectURL(blob) })),
            });
          }
        }
      },
    });
  }

  toggleNuevaGarantia() {
    this.mostrarNueva.update(v => !v);
    this.nuevaError.set(null);
    if (this.mostrarNueva()) this.resetNueva();
  }

  onSubtipoChange() {
    this.nuevoDatos = {};
    this.nuevaError.set(null);
  }

  private resetNueva() {
    this.nuevoSubtipo = 'vehiculo';
    this.nuevoValor = null;
    this.nuevoDescripcion = '';
    this.nuevoDatos = {};
  }

  crearGarantia() {
    const id = this.prestamo()?.id;
    if (!id) return;
    // Normaliza numéricos (año, superficie, cantidad) según el catálogo.
    const datos: Record<string, any> = {};
    for (const campo of this.camposActuales()) {
      const v = this.nuevoDatos[campo.key];
      if (v === undefined || v === null || v === '') continue;
      datos[campo.key] = campo.type === 'number' ? Number(v) : v;
    }
    this.guardandoGarantia.set(true);
    this.nuevaError.set(null);
    this.loanSvc.createGarantia(id, {
      subtipo: this.nuevoSubtipo,
      descripcion: this.nuevoDescripcion || undefined,
      valor_estimado: this.nuevoValor ?? undefined,
      datos,
    }).subscribe({
      next: () => {
        this.guardandoGarantia.set(false);
        this.mostrarNueva.set(false);
        this.loadGarantias();
      },
      error: e => { this.nuevaError.set(e.error?.error || e.message); this.guardandoGarantia.set(false); },
    });
  }

  eliminarGarantia(gid: string) {
    const id = this.prestamo()?.id;
    if (!id) return;
    this.loanSvc.deleteGarantia(id, gid).subscribe({ next: () => this.loadGarantias() });
  }

  onFileGarantia(ev: Event, gid: string) {
    const input = ev.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = ''; // permite re-seleccionar el mismo archivo
    const id = this.prestamo()?.id;
    if (!file || !id) return;
    this.subiendoEn.set(gid);
    this.loanSvc.uploadImagen(id, gid, file).subscribe({
      next: () => { this.subiendoEn.set(null); this.loadGarantias(); },
      error: () => { this.subiendoEn.set(null); },
    });
  }

  eliminarImagen(gid: string, iid: string) {
    const id = this.prestamo()?.id;
    if (!id) return;
    this.loanSvc.deleteImagen(id, gid, iid).subscribe({ next: () => this.loadGarantias() });
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
