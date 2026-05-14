import { Component } from '@angular/core';

@Component({
  selector: 'app-prestamos',
  template: `
    <h2>Préstamos</h2>
    <p>Módulo placeholder — consume <code>loan-service</code> vía gateway.</p>
  `,
  styles: [`
    h2 { margin-top: 0; color: #2d3748; }
    code { background: #edf2f7; padding: 2px 6px; border-radius: 4px; }
  `],
})
export class Prestamos {}
