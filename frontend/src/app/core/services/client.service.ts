import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';

export interface Cliente {
  id: string;
  nombres: string;
  apellidos: string;
  ci: string;
  fecha_nacimiento: string;
  telefono?: string;
  direccion?: string;
  email?: string;
  estado: 'activo' | 'inactivo' | 'bloqueado';
  foto_url?: string;
  created_at: string;
  updated_at: string;
}

export interface ListResult<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}

export interface CreateClienteInput {
  nombres: string;
  apellidos: string;
  ci: string;
  fecha_nacimiento: string;
  telefono?: string;
  direccion?: string;
  email?: string;
}

@Injectable({ providedIn: 'root' })
export class ClienteService {
  private http = inject(HttpClient);
  private base = '/api/clients';

  list(opts: { page?: number; limit?: number; search?: string } = {}): Observable<ListResult<Cliente>> {
    const params: Record<string, string> = {};
    if (opts.page != null) params['page'] = String(opts.page);
    if (opts.limit != null) params['limit'] = String(opts.limit);
    if (opts.search) params['search'] = opts.search;
    return this.http.get<ListResult<Cliente>>(this.base, { params });
  }

  get(id: string): Observable<Cliente> {
    return this.http.get<Cliente>(`${this.base}/${id}`);
  }

  create(data: CreateClienteInput): Observable<Cliente> {
    return this.http.post<Cliente>(this.base, data);
  }

  update(id: string, data: Partial<CreateClienteInput>): Observable<Cliente> {
    return this.http.put<Cliente>(`${this.base}/${id}`, data);
  }

  delete(id: string): Observable<void> {
    return this.http.delete<void>(`${this.base}/${id}`);
  }
}
