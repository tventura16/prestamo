import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DocumentService, Documento } from '../../core/services/document.service';

@Component({
  selector: 'app-documentos',
  imports: [CommonModule, FormsModule],
  template: `
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
      <h2 class="m-0 text-xl font-semibold text-ink">Documentos</h2>
      <div class="flex flex-wrap items-center gap-2">
        <select [(ngModel)]="tipo" (change)="reload()" class="ui-input w-full max-w-xs">
          <option value="">Todos los tipos</option>
          <option value="contrato">Contrato</option>
          <option value="plan_pagos">Plan de pagos</option>
          <option value="recibo">Recibo</option>
          <option value="estado_cuenta">Estado de cuenta</option>
          <option value="carta_mora">Carta de mora</option>
        </select>
        <span class="flex flex-wrap items-center gap-2">
          <input [(ngModel)]="clienteIdGen" placeholder="cliente_id (UUID)" class="ui-input w-full max-w-xs">
          <button (click)="generarEstado()" [disabled]="generando() || !clienteIdGen.trim()"
                  class="rounded-md bg-navy px-4 py-2 text-sm font-medium text-white transition hover:bg-navy-light disabled:cursor-not-allowed disabled:opacity-50">
            {{ generando() ? 'Generando...' : 'Generar estado de cuenta' }}
          </button>
        </span>
      </div>
    </div>

    @if (loading()) { <p class="mb-2 text-sm text-muted">Cargando...</p> }
    @if (error()) { <p class="mb-2 rounded-md bg-red-50 p-3 text-sm text-red-600">{{ error() }}</p> }

    <div class="overflow-x-auto rounded-lg bg-white shadow-sm">
      <table class="w-full min-w-[720px] border-collapse text-sm">
        <thead>
          <tr class="border-b border-slate-200 bg-slate-50 text-left text-slate-600">
            <th class="px-3 py-2 font-semibold">Tipo</th>
            <th class="px-3 py-2 font-semibold">Archivo</th>
            <th class="px-3 py-2 font-semibold">Referencia</th>
            <th class="px-3 py-2 text-right font-semibold">Tamaño</th>
            <th class="px-3 py-2 font-semibold">Estado</th>
            <th class="px-3 py-2 font-semibold">Generado</th>
            <th class="px-3 py-2"></th>
          </tr>
        </thead>
        <tbody>
          @for (d of items(); track d.id) {
            <tr class="border-b border-slate-100 last:border-0">
              <td class="px-3 py-2"><span class="rounded-full px-2.5 py-0.5 text-xs font-medium capitalize" [class]="tipoBadge(d.tipo)">{{ tipoLabel(d.tipo) }}</span></td>
              <td class="px-3 py-2">{{ d.nombre_archivo }}</td>
              <td class="px-3 py-2 space-x-1">
                @if (d.prestamo_id) { <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs" title="préstamo">P:{{ d.prestamo_id | slice:0:8 }}</code> }
                @if (d.cliente_id) { <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs" title="cliente">C:{{ d.cliente_id | slice:0:8 }}</code> }
                @if (d.pago_id) { <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs" title="pago">$:{{ d.pago_id | slice:0:8 }}</code> }
              </td>
              <td class="px-3 py-2 text-right text-muted">{{ d.tamanio_kb ? d.tamanio_kb + ' KB' : '—' }}</td>
              <td class="px-3 py-2"><span class="rounded-full px-2.5 py-0.5 text-xs font-medium capitalize" [class]="badge(d.estado)">{{ d.estado }}</span></td>
              <td class="px-3 py-2 text-muted">{{ d.generado_at | slice:0:10 }} {{ d.generado_at | slice:11:16 }}</td>
              <td class="px-3 py-2">
                <button (click)="descargar(d)" [disabled]="descargando() === d.id"
                        class="text-navy-light hover:underline disabled:cursor-not-allowed disabled:text-muted disabled:no-underline">
                  {{ descargando() === d.id ? '...' : 'Descargar' }}
                </button>
              </td>
            </tr>
          } @empty {
            <tr><td colspan="7" class="px-3 py-6 text-center text-muted">Sin documentos</td></tr>
          }
        </tbody>
      </table>
    </div>

    <p class="mt-2 text-sm text-muted">Mostrando {{ items().length }} de {{ total() }} documentos</p>
  `,
})
export class Documentos implements OnInit {
  private svc = inject(DocumentService);

  items = signal<Documento[]>([]);
  total = signal(0);
  loading = signal(false);
  error = signal<string | null>(null);
  descargando = signal<string | null>(null);
  generando = signal(false);
  tipo = '';
  clienteIdGen = '';

  ngOnInit() { this.reload(); }

  reload() {
    this.loading.set(true);
    this.error.set(null);
    this.svc.list({ limit: 100, tipo: this.tipo || undefined }).subscribe({
      next: r => { this.items.set(r.items); this.total.set(r.total); this.loading.set(false); },
      error: e => { this.error.set(e.error?.error || e.message); this.loading.set(false); },
    });
  }

  descargar(d: Documento) {
    this.descargando.set(d.id);
    this.svc.download(d.id).subscribe({
      next: blob => {
        this.descargando.set(null);
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = d.nombre_archivo || `documento-${d.id}.pdf`;
        a.click();
        URL.revokeObjectURL(url);
      },
      error: e => { this.error.set(e.error?.error || e.message); this.descargando.set(null); },
    });
  }

  generarEstado() {
    this.generando.set(true);
    this.error.set(null);
    this.svc.generateStatement(this.clienteIdGen.trim()).subscribe({
      next: () => { this.generando.set(false); this.clienteIdGen = ''; this.reload(); },
      error: e => { this.error.set(e.error?.error || e.message); this.generando.set(false); },
    });
  }

  tipoLabel(t: string): string {
    return { contrato: 'Contrato', plan_pagos: 'Plan de pagos', recibo: 'Recibo',
      estado_cuenta: 'Estado de cuenta', carta_mora: 'Carta de mora' }[t] ?? t;
  }

  // Color de badge por estado de generación del documento.
  badge(estado: string): string {
    switch (estado) {
      case 'generado': return 'bg-green-100 text-green-800';
      case 'error': return 'bg-red-100 text-red-800';
      case 'pendiente': return 'bg-orange-100 text-orange-800';
      case 'enviado': return 'bg-slate-200 text-slate-700';
      default: return 'bg-slate-200 text-slate-600';
    }
  }

  // Color de badge por tipo de documento (preserva la paleta original por tipo).
  tipoBadge(tipo: string): string {
    switch (tipo) {
      case 'contrato': return 'bg-green-100 text-green-800';
      case 'plan_pagos': return 'bg-blue-100 text-blue-800';
      case 'recibo': return 'bg-orange-100 text-orange-800';
      case 'estado_cuenta': return 'bg-purple-100 text-purple-800';
      case 'carta_mora': return 'bg-red-100 text-red-800';
      default: return 'bg-slate-200 text-slate-600';
    }
  }
}
