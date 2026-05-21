import {
  ApplicationConfig,
  provideZoneChangeDetection,
  provideAppInitializer,
  inject,
} from '@angular/core';
import { provideRouter } from '@angular/router';
import { provideHttpClient, withInterceptors } from '@angular/common/http';

import { routes } from './app.routes';
import { KeycloakService } from './core/keycloak.service';
import { authInterceptor } from './core/auth.interceptor';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(routes),
    provideHttpClient(withInterceptors([authInterceptor])),

    // Inicializa Keycloak ANTES de bootstrap. Si no hay sesión,
    // redirige al login. Bloquea hasta tener un token válido.
    provideAppInitializer(() => {
      const kc = inject(KeycloakService);
      return kc.init();
    }),
  ],
};
