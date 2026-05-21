import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ClienteService, Cliente, CreateClienteInput } from '../../core/services/client.service';

@Component({
  selector: 'app-clientes',
  imports: [CommonModule, FormsModule],
  template: `
    <div class="header">
      <h2>Clientes</h2>
      <button class="btn primary" (click)="toggleForm()">
        {{ showForm() ? 'Cancelar' : '+ Nuevo cliente' }}
      </button>
    </div>

    @if (showForm()) {
      <form class="form-card" (ngSubmit)="onSubmit()" #f="ngForm">
        <div class="row">
          <label>Nombres *
            <input [(ngModel)]="form.nombres" name="nombres" required maxlength="100">
          </label>
          <label>Apellidos *
            <input [(ngModel)]="form.apellidos" name="apellidos" required maxlength="100">
          </label>
        </div>
        <div class="row">
          <label>CI *
            <input [(ngModel)]="form.ci" name="ci" required minlength="4" maxlength="20">
          </label>
          <label>Fecha nacimiento *
            <input type="date" [(ngModel)]="form.fecha_nacimiento" name="fecha_nacimiento" required>
          </label>
        </div>
        <div class="row">
          <label>Teléfono
            <input [(ngModel)]="form.telefono" name="telefono" maxlength="20">
          </label>
          <label>Email
            <input type="email" [(ngModel)]="form.email" name="email">
          </label>
        </div>
        <label class="full">Dirección
          <input [(ngModel)]="form.direccion" name="direccion">
        </label>

        @if (submitError()) {
          <p class="err">{{ submitError() }}</p>
        }

        <div class="actions">
          <button type="button" class="btn" (click)="toggleForm()">Cancelar</button>
          <button type="submit" class="btn primary" [disabled]="!f.valid || submitting()">
            {{ submitting() ? 'Guardando...' : 'Guardar' }}
          </button>
        </div>
      </form>
    }

    <div class="search">
      <input type="search" [(ngModel)]="search" placeholder="Buscar por nombre o CI..."
             (input)="onSearch()" name="search">
      @if (loading()) { <span class="hint">cargando...</span> }
    </div>

    @if (error()) {
      <p class="err">{{ error() }}</p>
    }

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Nombre</th>
            <th>CI</th>
            <th>Teléfono</th>
            <th>Email</th>
            <th>Estado</th>
            <th>Creado</th>
          </tr>
        </thead>
        <tbody>
          @for (c of items(); track c.id) {
            <tr>
              <td>{{ c.nombres }} {{ c.apellidos }}</td>
              <td><code>{{ c.ci }}</code></td>
              <td>{{ c.telefono || '—' }}</td>
              <td>{{ c.email || '—' }}</td>
              <td><span class="badge" [class]="'b-' + c.estado">{{ c.estado }}</span></td>
              <td class="muted">{{ c.created_at | slice:0:10 }}</td>
            </tr>
          } @empty {
            <tr><td colspan="6" class="muted center">Sin clientes</td></tr>
          }
        </tbody>
      </table>
    </div>
    <p class="hint">Mostrando {{ items().length }} de {{ total() }} clientes</p>
  `,
  styles: [`
    .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
    h2 { margin: 0; color: #2d3748; }
    .form-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin-bottom: 16px; display: flex; flex-direction: column; gap: 12px; }
    .row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
    label { display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: #4a5568; }
    label.full { width: 100%; }
    input { padding: 8px 10px; border: 1px solid #cbd5e0; border-radius: 6px; font-size: 14px; }
    input:focus { outline: none; border-color: #2c5282; }
    .actions { display: flex; gap: 8px; justify-content: flex-end; }
    .btn { padding: 8px 16px; border: 1px solid #cbd5e0; background: white; border-radius: 6px; cursor: pointer; font-size: 14px; }
    .btn.primary { background: #1a365d; color: white; border-color: #1a365d; }
    .btn.primary:disabled { background: #a0aec0; border-color: #a0aec0; cursor: not-allowed; }
    .search { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
    .search input { flex: 1; max-width: 320px; }
    .table-wrap { background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); overflow: hidden; }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 10px 14px; text-align: left; font-size: 14px; }
    th { background: #f7fafc; color: #4a5568; font-weight: 600; border-bottom: 1px solid #e2e8f0; }
    td { border-bottom: 1px solid #edf2f7; }
    tr:last-child td { border-bottom: none; }
    code { background: #edf2f7; padding: 2px 6px; border-radius: 4px; font-size: 13px; }
    .badge { padding: 2px 10px; border-radius: 12px; font-size: 12px; font-weight: 500; }
    .b-activo { background: #c6f6d5; color: #22543d; }
    .b-inactivo { background: #e2e8f0; color: #4a5568; }
    .b-bloqueado { background: #fed7d7; color: #742a2a; }
    .muted { color: #718096; }
    .center { text-align: center; }
    .hint { color: #718096; font-size: 13px; margin-top: 8px; }
    .err { color: #c53030; background: #fff5f5; padding: 10px; border-radius: 6px; margin: 8px 0; }
  `],
})
export class Clientes implements OnInit {
  private svc = inject(ClienteService);

  items = signal<Cliente[]>([]);
  total = signal(0);
  loading = signal(false);
  error = signal<string | null>(null);

  showForm = signal(false);
  submitting = signal(false);
  submitError = signal<string | null>(null);
  form: CreateClienteInput = this.emptyForm();
  search = '';
  private searchTimer: any;

  ngOnInit() {
    this.load();
  }

  load() {
    this.loading.set(true);
    this.error.set(null);
    this.svc.list({ limit: 50, search: this.search }).subscribe({
      next: r => { this.items.set(r.items); this.total.set(r.total); this.loading.set(false); },
      error: e => { this.error.set(e.error?.error || e.message); this.loading.set(false); },
    });
  }

  onSearch() {
    clearTimeout(this.searchTimer);
    this.searchTimer = setTimeout(() => this.load(), 300);
  }

  toggleForm() {
    this.showForm.update(v => !v);
    this.submitError.set(null);
    if (!this.showForm()) this.form = this.emptyForm();
  }

  onSubmit() {
    this.submitting.set(true);
    this.submitError.set(null);
    this.svc.create(this.form).subscribe({
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

  private emptyForm(): CreateClienteInput {
    return {
      nombres: '', apellidos: '', ci: '', fecha_nacimiento: '',
      telefono: '', direccion: '', email: '',
    };
  }
}
