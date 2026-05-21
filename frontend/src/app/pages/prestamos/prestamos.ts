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
    <div class="header">
      <h2>Préstamos</h2>
      <div class="filters">
        <select [(ngModel)]="estadoFilter" (change)="load()" name="estado">
          <option value="">Todos los estados</option>
          <option value="pendiente">Pendiente</option>
          <option value="aprobado">Aprobado</option>
          <option value="activo">Activo</option>
          <option value="mora">Mora</option>
          <option value="finalizado">Finalizado</option>
          <option value="rechazado">Rechazado</option>
        </select>
        <button class="btn primary" (click)="toggleForm()">
          {{ showForm() ? 'Cancelar' : '+ Nuevo préstamo' }}
        </button>
      </div>
    </div>

    @if (showForm()) {
      <form class="form-card" (ngSubmit)="onSubmit()" #f="ngForm">
        <h3>Nueva solicitud</h3>
        <div class="row">
          <label>Cliente *
            <select [(ngModel)]="form.cliente_id" name="cliente_id" required>
              <option value="" disabled>Seleccione...</option>
              @for (c of clientes(); track c.id) {
                <option [value]="c.id">{{ c.nombres }} {{ c.apellidos }} — CI {{ c.ci }}</option>
              }
            </select>
          </label>
          <label>Frecuencia *
            <select [(ngModel)]="form.frecuencia" name="frecuencia" required>
              <option value="diaria">Diaria</option>
              <option value="semanal">Semanal</option>
              <option value="quincenal">Quincenal</option>
              <option value="mensual">Mensual</option>
            </select>
          </label>
        </div>
        <div class="row">
          <label>Monto solicitado (BOB) *
            <input type="number" min="1" step="0.01"
                   [(ngModel)]="form.monto_solicitado" name="monto" required>
          </label>
          <label>Tasa de interés (%) por periodo *
            <input type="number" min="0" step="0.01"
                   [(ngModel)]="form.tasa_porcentaje" name="tasa" required>
          </label>
          <label>Número de cuotas *
            <input type="number" min="1" max="600" step="1"
                   [(ngModel)]="form.num_cuotas" name="num_cuotas" required>
          </label>
        </div>
        <label class="full">Observaciones
          <input [(ngModel)]="form.observaciones" name="observaciones">
        </label>

        @if (submitError()) { <p class="err">{{ submitError() }}</p> }

        <div class="actions">
          <button type="button" class="btn" (click)="toggleForm()">Cancelar</button>
          <button type="submit" class="btn primary"
                  [disabled]="!f.valid || submitting()">
            {{ submitting() ? 'Guardando...' : 'Crear solicitud' }}
          </button>
        </div>
      </form>
    }

    @if (loading()) { <p class="hint">Cargando...</p> }
    @if (error()) { <p class="err">{{ error() }}</p> }

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Cliente</th>
            <th>Monto</th>
            <th>Tasa</th>
            <th>Cuotas</th>
            <th>Frecuencia</th>
            <th>Estado</th>
            <th>Solicitud</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          @for (p of items(); track p.id) {
            <tr>
              <td><code>{{ p.id | slice:0:8 }}</code></td>
              <td>{{ clienteName(p.cliente_id) }}</td>
              <td>{{ (p.monto_aprobado ?? p.monto_solicitado) | currency:'BOB':'symbol-narrow':'1.2-2' }}</td>
              <td>{{ (p.tasa_interes * 100).toFixed(2) }}%</td>
              <td>{{ p.num_cuotas }}</td>
              <td>{{ p.frecuencia }}</td>
              <td><span class="badge" [class]="'b-' + p.estado">{{ p.estado }}</span></td>
              <td class="muted">{{ p.fecha_solicitud | slice:0:10 }}</td>
              <td><a [routerLink]="['/prestamos', p.id]" class="link">ver →</a></td>
            </tr>
          } @empty {
            <tr><td colspan="9" class="muted center">Sin préstamos</td></tr>
          }
        </tbody>
      </table>
    </div>

    <p class="hint">Mostrando {{ items().length }} de {{ total() }} préstamos</p>
  `,
  styles: [`
    .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
    .filters { display: flex; gap: 8px; }
    h2 { margin: 0; color: #2d3748; }
    h3 { margin: 0 0 12px; color: #2d3748; }
    select, input { padding: 8px 12px; border: 1px solid #cbd5e0; border-radius: 6px; font-size: 14px; }
    select:focus, input:focus { outline: none; border-color: #2c5282; }
    .form-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin-bottom: 16px; display: flex; flex-direction: column; gap: 12px; }
    .row { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 12px; }
    .row label { display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: #4a5568; }
    label.full { display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: #4a5568; }
    .actions { display: flex; gap: 8px; justify-content: flex-end; }
    .btn { padding: 8px 16px; border: 1px solid #cbd5e0; background: white; border-radius: 6px; cursor: pointer; font-size: 14px; }
    .btn.primary { background: #1a365d; color: white; border-color: #1a365d; }
    .btn.primary:disabled { background: #a0aec0; border-color: #a0aec0; cursor: not-allowed; }
    .table-wrap { background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); overflow: hidden; }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 10px 14px; text-align: left; font-size: 14px; }
    th { background: #f7fafc; color: #4a5568; font-weight: 600; border-bottom: 1px solid #e2e8f0; }
    td { border-bottom: 1px solid #edf2f7; }
    tr:last-child td { border-bottom: none; }
    code { background: #edf2f7; padding: 2px 6px; border-radius: 4px; font-size: 12px; }
    .badge { padding: 2px 10px; border-radius: 12px; font-size: 12px; font-weight: 500; }
    .b-pendiente { background: #feebc8; color: #7b341e; }
    .b-aprobado { background: #bee3f8; color: #2a4365; }
    .b-activo { background: #c6f6d5; color: #22543d; }
    .b-finalizado { background: #e2e8f0; color: #4a5568; }
    .b-mora { background: #fed7d7; color: #742a2a; }
    .b-rechazado { background: #fed7d7; color: #742a2a; }
    .muted { color: #718096; }
    .center { text-align: center; }
    .hint { color: #718096; font-size: 13px; margin-top: 8px; }
    .err { color: #c53030; background: #fff5f5; padding: 10px; border-radius: 6px; }
    .link { color: #2c5282; text-decoration: none; font-weight: 500; }
    .link:hover { text-decoration: underline; }
  `],
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
