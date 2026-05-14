import { Component } from '@angular/core';

@Component({
  selector: 'app-clientes',
  template: `
    <h2>Clientes</h2>
    <p>Módulo placeholder — consume <code>client-service</code> vía gateway.</p>
  `,
  styles: [`
    h2 { margin-top: 0; color: #2d3748; }
    code { background: #edf2f7; padding: 2px 6px; border-radius: 4px; }
  `],
})
export class Clientes {}
