import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule, CurrencyPipe } from '@angular/common';
import { PaymentService, Pago } from '../../core/services/payment.service';
import { DocumentService } from '../../core/services/document.service';

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
              <td>
                <button class="link-btn" (click)="generarRecibo(p)" [disabled]="generando() === p.id">
                  {{ generando() === p.id ? '...' : 'Recibo PDF' }}
                </button>
              </td>
            </tr>
          } @empty {
            <tr><td colspan="11" class="muted center">Sin pagos registrados</td></tr>
          }
        </tbody>
      </table>
    </div>

    <p class="hint">Mostrando {{ items().length }} de {{ total() }} pagos</p>
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
    tr.anulado { opacity: 0.5; text-decoration: line-through; }
    .link-btn { background: none; border: none; color: #2c5282; cursor: pointer; font-size: 13px; padding: 0; }
    .link-btn:hover { text-decoration: underline; }
    .link-btn:disabled { color: #a0aec0; cursor: not-allowed; }
  `],
})
export class Pagos implements OnInit {
  private svc = inject(PaymentService);
  private docSvc = inject(DocumentService);

  items = signal<Pago[]>([]);
  total = signal(0);
  loading = signal(false);
  error = signal<string | null>(null);
  generando = signal<string | null>(null);

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
}
