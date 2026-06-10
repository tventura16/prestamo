import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';

import {
  LoanService,
  Prestamo,
  EstadoPrestamo,
  Frecuencia,
  CreatePrestamoInput,
} from '../../core/services/loan.service';
import { ClienteService, Cliente } from '../../core/services/client.service';

interface PrestamoForm {
  cliente_id: string;
  monto_solicitado: number | null;
  tasa_porcentaje: number | null; // user enters 5 → API gets 0.05
  num_cuotas: number | null;
  frecuencia: Frecuencia;
  observaciones: string;
}

@Component({
  selector: 'app-prestamos',
  imports: [CommonModule, CurrencyPipe, RouterLink, FormsModule],
  template: `
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <h2 class="m-0 text-xl font-semibold text-ink">Préstamos</h2>
      <div class="flex flex-wrap items-center gap-2">
        <select [(ngModel)]="estadoFilter" (change)="load()" name="estado" class="ui-input">
          <option value="">Todos los estados</option>
          <option value="pendiente">Pendiente</option>
          <option value="aprobado">Aprobado</option>
          <option value="activo">Activo</option>
          <option value="mora">Mora</option>
          <option value="finalizado">Finalizado</option>
          <option value="rechazado">Rechazado</option>
        </select>
        <button (click)="toggleForm()"
                class="rounded-md bg-navy px-4 py-2 text-sm font-medium text-white transition hover:bg-navy-light">
          {{ showForm() ? 'Cancelar' : '+ Nuevo préstamo' }}
        </button>
      </div>
    </div>

    @if (showForm()) {
      <form (ngSubmit)="onSubmit()" #f="ngForm"
            class="mb-4 flex flex-col gap-3 rounded-lg bg-white p-5 shadow-sm">
        <h3 class="m-0 text-base font-semibold text-ink">Nueva solicitud</h3>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <label class="flex flex-col gap-1 text-sm text-slate-600">Cliente *
            <select [(ngModel)]="form.cliente_id" name="cliente_id" required class="ui-input">
              <option value="" disabled>Seleccione...</option>
              @for (c of clientes(); track c.id) {
                <option [value]="c.id">{{ c.nombres }} {{ c.apellidos }} — CI {{ c.ci }}</option>
              }
            </select>
          </label>
          <label class="flex flex-col gap-1 text-sm text-slate-600">Frecuencia *
            <select [(ngModel)]="form.frecuencia" name="frecuencia" required class="ui-input">
              <option value="diaria">Diaria</option>
              <option value="semanal">Semanal</option>
              <option value="quincenal">Quincenal</option>
              <option value="mensual">Mensual</option>
            </select>
          </label>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <label class="flex flex-col gap-1 text-sm text-slate-600">Monto solicitado (BOB) *
            <input type="number" min="1" step="0.01"
                   [(ngModel)]="form.monto_solicitado" name="monto" required class="ui-input">
          </label>
          <label class="flex flex-col gap-1 text-sm text-slate-600">Tasa de interés (%) por periodo *
            <input type="number" min="0" step="0.01"
                   [(ngModel)]="form.tasa_porcentaje" name="tasa" required class="ui-input">
          </label>
          <label class="flex flex-col gap-1 text-sm text-slate-600">Número de cuotas *
            <input type="number" min="1" max="600" step="1"
                   [(ngModel)]="form.num_cuotas" name="num_cuotas" required class="ui-input">
          </label>
        </div>
        <label class="flex flex-col gap-1 text-sm text-slate-600">Observaciones
          <input [(ngModel)]="form.observaciones" name="observaciones" class="ui-input">
        </label>

        @if (submitError()) {
          <p class="rounded-md bg-red-50 p-3 text-sm text-red-600">{{ submitError() }}</p>
        }

        <div class="flex flex-wrap justify-end gap-2">
          <button type="button" (click)="toggleForm()"
                  class="rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-50">Cancelar</button>
          <button type="submit" [disabled]="!f.valid || submitting()"
                  class="rounded-md bg-navy px-4 py-2 text-sm font-medium text-white transition hover:bg-navy-light disabled:cursor-not-allowed disabled:opacity-50">
            {{ submitting() ? 'Guardando...' : 'Crear solicitud' }}
          </button>
        </div>
      </form>
    }

    @if (loading()) { <p class="mb-2 text-sm text-muted">Cargando...</p> }
    @if (error()) { <p class="mb-2 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ error() }}</p> }

    <div class="overflow-x-auto rounded-lg bg-white shadow-sm">
      <table class="w-full min-w-[820px] border-collapse text-sm">
        <thead>
          <tr class="border-b border-slate-200 bg-slate-50 text-left text-slate-600">
            <th class="px-3 py-2 font-semibold">ID</th>
            <th class="px-3 py-2 font-semibold">Cliente</th>
            <th class="px-3 py-2 text-right font-semibold">Monto</th>
            <th class="px-3 py-2 text-right font-semibold">Tasa</th>
            <th class="px-3 py-2 text-center font-semibold">Cuotas</th>
            <th class="px-3 py-2 font-semibold">Frecuencia</th>
            <th class="px-3 py-2 font-semibold">Estado</th>
            <th class="px-3 py-2 font-semibold">Solicitud</th>
            <th class="px-3 py-2"></th>
          </tr>
        </thead>
        <tbody>
          @for (p of items(); track p.id) {
            <tr class="border-b border-slate-100 last:border-0">
              <td class="px-3 py-2"><code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">{{ p.id | slice:0:8 }}</code></td>
              <td class="px-3 py-2">{{ clienteName(p.cliente_id) }}</td>
              <td class="px-3 py-2 text-right">{{ (p.monto_aprobado ?? p.monto_solicitado) | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
              <td class="px-3 py-2 text-right">{{ (p.tasa_interes * 100).toFixed(2) }}%</td>
              <td class="px-3 py-2 text-center">{{ p.num_cuotas }}</td>
              <td class="px-3 py-2 capitalize">{{ p.frecuencia }}</td>
              <td class="px-3 py-2"><span class="rounded-full px-2.5 py-0.5 text-xs font-medium capitalize" [class]="badge(p.estado)">{{ p.estado }}</span></td>
              <td class="px-3 py-2 text-muted">{{ p.fecha_solicitud | slice:0:10 }}</td>
              <td class="px-3 py-2"><a [routerLink]="['/prestamos', p.id]" class="font-semibold text-navy-light hover:underline">ver →</a></td>
            </tr>
          } @empty {
            <tr><td colspan="9" class="px-3 py-6 text-center text-muted">Sin préstamos</td></tr>
          }
        </tbody>
      </table>
    </div>

    <p class="mt-2 text-sm text-muted">Mostrando {{ items().length }} de {{ total() }} préstamos</p>
  `,
})
export class Prestamos implements OnInit {
  private svc = inject(LoanService);
  private clienteSvc = inject(ClienteService);

  items = signal<Prestamo[]>([]);
  total = signal(0);
  loading = signal(false);
  error = signal<string | null>(null);
  estadoFilter: EstadoPrestamo | '' = '';

  clientes = signal<Cliente[]>([]);
  showForm = signal(false);
  submitting = signal(false);
  submitError = signal<string | null>(null);
  form: PrestamoForm = this.emptyForm();

  // Color de badge por estado de préstamo.
  badge(estado: string): string {
    switch (estado) {
      case 'activo':
        return 'bg-green-100 text-green-800';
      case 'mora':
      case 'rechazado':
        return 'bg-red-100 text-red-800';
      case 'finalizado':
        return 'bg-slate-200 text-slate-700';
      case 'pendiente':
      case 'aprobado':
        return 'bg-orange-100 text-orange-800';
      default:
        return 'bg-slate-200 text-slate-600';
    }
  }

  ngOnInit() {
    this.load();
    // Cargar clientes para el selector del formulario.
    this.clienteSvc.list({ limit: 200 }).subscribe({
      next: r => this.clientes.set(r.items),
    });
  }

  load() {
    this.loading.set(true);
    this.error.set(null);
    this.svc.list({
      limit: 50,
      estado: this.estadoFilter || undefined,
    }).subscribe({
      next: r => { this.items.set(r.items); this.total.set(r.total); this.loading.set(false); },
      error: e => { this.error.set(e.error?.error || e.message); this.loading.set(false); },
    });
  }

  toggleForm() {
    this.showForm.update(v => !v);
    this.submitError.set(null);
    if (!this.showForm()) this.form = this.emptyForm();
  }

  onSubmit() {
    if (this.form.monto_solicitado == null || this.form.tasa_porcentaje == null || this.form.num_cuotas == null) {
      return;
    }
    const payload: CreatePrestamoInput = {
      cliente_id: this.form.cliente_id,
      monto_solicitado: this.form.monto_solicitado,
      tasa_interes: this.form.tasa_porcentaje / 100, // 5 → 0.05
      num_cuotas: this.form.num_cuotas,
      frecuencia: this.form.frecuencia,
      observaciones: this.form.observaciones || undefined,
    };
    this.submitting.set(true);
    this.submitError.set(null);
    this.svc.create(payload).subscribe({
      next: () => {
        this.submitting.set(false);
        this.showForm.set(false);
        this.form = this.emptyForm();
        this.load();
      },
      error: e => {
        this.submitError.set(e.error?.error || e.message);
        this.submitting.set(false);
      },
    });
  }

  clienteName(id: string): string {
    const c = this.clientes().find(c => c.id === id);
    return c ? `${c.nombres} ${c.apellidos}` : id.slice(0, 8) + '…';
  }

  private emptyForm(): PrestamoForm {
    return {
      cliente_id: '', monto_solicitado: null, tasa_porcentaje: 5,
      num_cuotas: 6, frecuencia: 'mensual', observaciones: '',
    };
  }
}
