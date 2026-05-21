import { inject } from '@angular/core';
import { KeycloakService } from './keycloak.service';

export function getCurrentOperatorId(): string {
  const keycloak = inject(KeycloakService);
  return keycloak.userId();
}

export const CURRENT_OPERATOR_ID = '';
