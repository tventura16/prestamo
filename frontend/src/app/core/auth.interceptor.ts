import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { from, switchMap } from 'rxjs';
import { KeycloakService } from './keycloak.service';

/**
 * Agrega Authorization: Bearer <token> a peticiones /api/*
 * y refresca el token si está por expirar.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const kc = inject(KeycloakService);

  // Solo proteger llamadas al gateway (no a recursos estáticos ni Keycloak).
  if (!req.url.startsWith('/api/') && !req.url.startsWith('http')) {
    return next(req);
  }
  if (!kc.token) {
    return next(req);
  }

  return from(kc.updateToken(30)).pipe(
    switchMap(() => {
      const authReq = req.clone({
        setHeaders: { Authorization: `Bearer ${kc.token}` },
      });
      return next(authReq);
    }),
  );
};
