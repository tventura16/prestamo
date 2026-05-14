import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./pages/dashboard/dashboard').then((m) => m.Dashboard),
  },
  {
    path: 'clientes',
    loadComponent: () =>
      import('./pages/clientes/clientes').then((m) => m.Clientes),
  },
  {
    path: 'prestamos',
    loadComponent: () =>
      import('./pages/prestamos/prestamos').then((m) => m.Prestamos),
  },
  {
    path: '**',
    redirectTo: '',
  },
];
