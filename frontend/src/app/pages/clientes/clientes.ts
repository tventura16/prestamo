import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ClienteService, Cliente, CreateClienteInput } from '../../core/services/client.service';

@Component({
  selector: 'app-clientes',
  imports: [CommonModule, FormsModule, RouterLink],
  template: `
    <div class="mb-4 flex items-center justify-between gap-3">
      <h2 class="m-0 text-xl font-semibold text-ink">Clientes</h2>
      <button (click)="toggleForm()"
              class="rounded-md bg-navy px-4 py-2 text-sm font-medium text-white transition hover:bg-navy-light">
        {{ showForm() ? 'Cancelar' : '+ Nuevo cliente' }}
      </button>
    </div>

    @if (showForm()) {
      <form (ngSubmit)="onSubmit()" #f="ngForm"
            class="mb-4 flex flex-col gap-3 rounded-lg bg-white p-5 shadow-sm">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <label class="flex flex-col gap-1 text-sm text-slate-600">Nombres *
            <input [(ngModel)]="form.nombres" name="nombres" required maxlength="100" class="ui-input">
          </label>
          <label class="flex flex-col gap-1 text-sm text-slate-600">Apellidos *
            <input [(ngModel)]="form.apellidos" name="apellidos" required maxlength="100" class="ui-input">
          </label>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <label class="flex flex-col gap-1 text-sm text-slate-600">CI *
            <input [(ngModel)]="form.ci" name="ci" required minlength="4" maxlength="20" class="ui-input">
          </label>
          <label class="flex flex-col gap-1 text-sm text-slate-600">Fecha nacimiento *
            <input type="date" [(ngModel)]="form.fecha_nacimiento" name="fecha_nacimiento" required class="ui-input">
          </label>
        </div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <label class="flex flex-col gap-1 text-sm text-slate-600">Teléfono
            <input [(ngModel)]="form.telefono" name="telefono" maxlength="20" class="ui-input">
          </label>
          <label class="flex flex-col gap-1 text-sm text-slate-600">Email
            <input type="email" [(ngModel)]="form.email" name="email" class="ui-input">
          </label>
        </div>
        <label class="flex flex-col gap-1 text-sm text-slate-600">Dirección
          <input [(ngModel)]="form.direccion" name="direccion" class="ui-input">
        </label>

        @if (submitError()) {
          <p class="rounded-md bg-red-50 p-3 text-sm text-red-600">{{ submitError() }}</p>
        }

        <div class="flex justify-end gap-2">
          <button type="button" (click)="toggleForm()"
                  class="rounded-md border border-slate-300 px-4 py-2 text-sm hover:bg-slate-50">Cancelar</button>
          <button type="submit" [disabled]="!f.valid || submitting()"
                  class="rounded-md bg-navy px-4 py-2 text-sm font-medium text-white transition hover:bg-navy-light disabled:cursor-not-allowed disabled:opacity-50">
            {{ submitting() ? 'Guardando...' : 'Guardar' }}
          </button>
        </div>
      </form>
    }

    <div class="mb-3 flex items-center gap-3">
      <input type="search" [(ngModel)]="search" placeholder="Buscar por nombre o CI..."
             (input)="onSearch()" name="search" class="ui-input w-full max-w-xs">
      @if (loading()) { <span class="text-sm text-muted">cargando...</span> }
    </div>

    @if (error()) {
      <p class="mb-2 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ error() }}</p>
    }

    <div class="overflow-x-auto rounded-lg bg-white shadow-sm">
      <table class="w-full min-w-[720px] border-collapse text-sm">
        <thead>
          <tr class="border-b border-slate-200 bg-slate-50 text-left text-slate-600">
            <th class="px-3 py-2 font-semibold">Nombre</th>
            <th class="px-3 py-2 font-semibold">CI</th>
            <th class="px-3 py-2 font-semibold">Teléfono</th>
            <th class="px-3 py-2 font-semibold">Email</th>
            <th class="px-3 py-2 font-semibold">Estado</th>
            <th class="px-3 py-2 font-semibold">Creado</th>
            <th class="px-3 py-2"></th>
          </tr>
        </thead>
        <tbody>
          @for (c of items(); track c.id) {
            <tr class="border-b border-slate-100 last:border-0">
              <td class="px-3 py-2">
                <a [routerLink]="['/clientes', c.id]" class="font-semibold text-navy-light hover:underline">{{ c.nombres }} {{ c.apellidos }}</a>
              </td>
              <td class="px-3 py-2"><code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">{{ c.ci }}</code></td>
              <td class="px-3 py-2">{{ c.telefono || '—' }}</td>
              <td class="px-3 py-2">{{ c.email || '—' }}</td>
              <td class="px-3 py-2"><span class="rounded-full px-2.5 py-0.5 text-xs font-medium" [class]="estadoBadge(c.estado)">{{ c.estado }}</span></td>
              <td class="px-3 py-2 text-muted">{{ c.created_at | slice:0:10 }}</td>
              <td class="px-3 py-2"><a [routerLink]="['/clientes', c.id]" class="font-semibold text-navy-light hover:underline">Historial →</a></td>
            </tr>
          } @empty {
            <tr><td colspan="7" class="px-3 py-6 text-center text-muted">Sin clientes</td></tr>
          }
        </tbody>
      </table>
    </div>
    <p class="mt-2 text-sm text-muted">Mostrando {{ items().length }} de {{ total() }} clientes</p>
  `,
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

  estadoBadge(estado: string): string {
    switch (estado) {
      case 'activo': return 'bg-green-100 text-green-800';
      case 'bloqueado': return 'bg-red-100 text-red-800';
      default: return 'bg-slate-200 text-slate-600';
    }
  }

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
