import Keycloak from 'keycloak-js';
import { writable } from 'svelte/store';

import {
	PUBLIC_KEYCLOAK_URL,
	PUBLIC_KEYCLOAK_REALM,
	PUBLIC_KEYCLOAK_CLIENT_ID
} from '$env/static/public';

export const keycloak = new Keycloak({
	url: PUBLIC_KEYCLOAK_URL,
	realm: PUBLIC_KEYCLOAK_REALM,
	clientId: PUBLIC_KEYCLOAK_CLIENT_ID
});

export const isAuthenticated = writable(false);
export const isInitialized = writable(false);

export async function initAuth(): Promise<boolean> {
	const authenticated = await keycloak.init({
		onLoad: 'check-sso',
		pkceMethod: 'S256',
		silentCheckSsoRedirectUri: window.location.origin + '/silent-check-sso.html'
	});
	isAuthenticated.set(authenticated);
	isInitialized.set(true);
	return authenticated;
}

export function login() {
	keycloak.login();
}

export function logout() {
	keycloak.logout({ redirectUri: window.location.origin });
}
