import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { DocumentService, Documento } from '../../core/services/document.service';

@Component({
  selector: 'app-documentos',
  imports: [CommonModule, FormsModule],
  template: `
    <div class="head">
      <h2>Documentos</h2>
      <div class="filters">
        <select [(ngModel)]="tipo" (change)="reload()">
          <option value="">Todos los tipos</option>
          <option value="contrato">Contrato</option>
          <option value="plan_pagos">Plan de pagos</option>
          <option value="recibo">Recibo</option>
          <option value="estado_cuenta">Estado de cuenta</option>
          <option value="carta_mora">Carta de mora</option>
        </select>
        <span class="gen">
          <input [(ngModel)]="clienteIdGen" placeholder="cliente_id (UUID)" />
          <button class="btn" (click)="generarEstado()" [disabled]="generando() || !clienteIdGen.trim()">
            {{ generando() ? 'Generando...' : 'Generar estado de cuenta' }}
          </button>
        </span>
      </div>
    </div>

    @if (loading()) { <p class="hint">Cargando...</p> }
    @if (error()) { <p class="err">{{ error() }}</p> }

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Tipo</th><th>Archivo</th><th>Referencia</th>
            <th class="r">Tamaño</th><th>Estado</th><th>Generado</th><th></th>
          </tr>
        </thead>
        <tbody>
          @for (d of items(); track d.id) {
            <tr>
              <td><span class="badge" [class]="'t-' + d.tipo">{{ tipoLabel(d.tipo) }}</span></td>
              <td>{{ d.nombre_archivo }}</td>
              <td>
                @if (d.prestamo_id) { <code title="préstamo">P:{{ d.prestamo_id | slice:0:8 }}</code> }
                @if (d.cliente_id) { <code title="cliente">C:{{ d.cliente_id | slice:0:8 }}</code> }
                @if (d.pago_id) { <code title="pago">$:{{ d.pago_id | slice:0:8 }}</code> }
              </td>
              <td class="r muted">{{ d.tamanio_kb ? d.tamanio_kb + ' KB' : '—' }}</td>
              <td><span class="badge estado">{{ d.estado }}</span></td>
              <td class="muted">{{ d.generado_at | slice:0:10 }} {{ d.generado_at | slice:11:16 }}</td>
              <td>
                <button class="link-btn" (click)="descargar(d)" [disabled]="descargando() === d.id">
                  {{ descargando() === d.id ? '...' : 'Descargar' }}
                </button>
              </td>
            </tr>
          } @empty {
            <tr><td colspan="7" class="muted center">Sin documentos</td></tr>
          }
        </tbody>
      </table>
    </div>

    <p class="hint">Mostrando {{ items().length }} de {{ total() }} documentos</p>
  `,
  styles: [`
    h2 { color: #2d3748; margin: 0; }
    .head { display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 12px; margin-bottom: 12px; }
    .filters { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
    .gen { display: flex; gap: 6px; }
    select, input { padding: 6px 10px; border: 1px solid #cbd5e0; border-radius: 6px; font-size: 13px; }
    input { width: 220px; }
    .btn { background: #2c5282; color: white; border: none; padding: 6px 12px; border-radius: 6px; cursor: pointer; font-size: 13px; }
    .btn:disabled { background: #a0aec0; cursor: not-allowed; }
    .table-wrap { background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); overflow: hidden; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    th, td { padding: 8px 12px; text-align: left; }
    th { background: #f7fafc; color: #4a5568; font-weight: 600; border-bottom: 1px solid #e2e8f0; }
    td { border-bottom: 1px solid #edf2f7; }
    .r { text-align: right; }
    code { background: #edf2f7; padding: 2px 6px; border-radius: 4px; font-size: 11px; margin-right: 4px; }
    .badge { padding: 2px 10px; border-radius: 12px; font-size: 12px; background: #e2e8f0; color: #2d3748; }
    .t-contrato { background: #c6f6d5; color: #22543d; }
    .t-plan_pagos { background: #bee3f8; color: #2a4365; }
    .t-recibo { background: #feebc8; color: #7b341e; }
    .t-estado_cuenta { background: #e9d8fd; color: #44337a; }
    .t-carta_mora { background: #fed7d7; color: #822727; }
    .estado { background: #edf2f7; color: #4a5568; }
    .muted { color: #718096; } .center { text-align: center; }
    .hint { color: #718096; font-size: 13px; margin-top: 8px; }
    .err { color: #c53030; background: #fff5f5; padding: 10px; border-radius: 6px; }
    .link-btn { background: none; border: none; color: #2c5282; cursor: pointer; font-size: 13px; padding: 0; }
    .link-btn:hover { text-decoration: underline; }
    .link-btn:disabled { color: #a0aec0; cursor: not-allowed; }
  `],
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
}
