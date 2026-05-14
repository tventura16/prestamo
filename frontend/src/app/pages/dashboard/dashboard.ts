import { Component } from '@angular/core';

@Component({
  selector: 'app-dashboard',
  template: `
    <h2>Dashboard</h2>
    <div class="cards">
      <div class="card">
        <span class="label">Préstamos activos</span>
        <span class="value">—</span>
      </div>
      <div class="card">
        <span class="label">Clientes registrados</span>
        <span class="value">—</span>
      </div>
      <div class="card">
        <span class="label">Cuotas vencidas</span>
        <span class="value">—</span>
      </div>
      <div class="card">
        <span class="label">Ingresos del mes</span>
        <span class="value">—</span>
      </div>
    </div>
    <p class="hint">Conectar con report-service vía API Gateway en <code>/api/reports</code>.</p>
  `,
  styles: [`
    h2 { margin-top: 0; color: #2d3748; }
    .cards {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 16px;
      margin-bottom: 24px;
    }
    .card {
      background: white;
      padding: 20px;
      border-radius: 8px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .label { color: #718096; font-size: 13px; }
    .value { font-size: 28px; font-weight: 600; color: #1a365d; }
    .hint { color: #718096; font-size: 13px; }
    code { background: #edf2f7; padding: 2px 6px; border-radius: 4px; }
  `],
})
export class Dashboard {}
