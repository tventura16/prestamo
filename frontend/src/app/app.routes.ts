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
    path: 'prestamos/:id',
    loadComponent: () =>
      import('./pages/prestamos/prestamo-detail').then((m) => m.PrestamoDetail),
  },
  {
    path: 'pagos',
    loadComponent: () =>
      import('./pages/pagos/pagos').then((m) => m.Pagos),
  },
  {
    path: '**',
    redirectTo: '',
  },
];
