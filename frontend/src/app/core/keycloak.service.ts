import { Injectable, signal } from '@angular/core';
import Keycloak from 'keycloak-js';

export interface KeycloakClaims {
  sub: string;
  preferred_username: string;
  email?: string;
  realm_access?: { roles: string[] };
}

@Injectable({ providedIn: 'root' })
export class KeycloakService {
  private kc!: Keycloak;

  readonly authenticated = signal(false);
  readonly username      = signal<string>('');
  readonly email         = signal<string>('');
  readonly userId        = signal<string>('');
  readonly roles         = signal<string[]>([]);

  async init(): Promise<void> {
    this.kc = new Keycloak({
      url:      'http://localhost:8080',
      realm:    'prestamos',
      clientId: 'prestamos-frontend',
    });

    const authenticated = await this.kc.init({
      onLoad:            'login-required',
      pkceMethod:        'S256',
      checkLoginIframe:  false,
      enableLogging:     true,
    });

    this.authenticated.set(authenticated);
    if (authenticated) {
      this.refreshClaims();
      // Auto-refresh cada 60s para mantener viva la sesión
      setInterval(() => this.kc.updateToken(70).catch(() => this.logout()), 60_000);
    }
  }

  private refreshClaims(): void {
    const c = this.kc.tokenParsed as KeycloakClaims | undefined;
    if (!c) return;
    this.username.set(c.preferred_username ?? '');
    this.email.set(c.email ?? '');
    this.userId.set(c.sub ?? '');
    this.roles.set(c.realm_access?.roles ?? []);
  }

  get token(): string | undefined {
    return this.kc?.token;
  }

  hasRole(role: string): boolean {
    return this.roles().includes(role);
  }

  /** Refresca el token si expira en menos de `minValidity` segundos. */
  async updateToken(minValidity = 30): Promise<void> {
    if (!this.kc) return;
    try {
      const refreshed = await this.kc.updateToken(minValidity);
      if (refreshed) this.refreshClaims();
    } catch {
      await this.logout();
    }
  }

  logout(): Promise<void> {
    return this.kc.logout({ redirectUri: window.location.origin });
  }
}
