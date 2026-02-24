import { useReducer } from 'react';
import {
  CredentialType,
  CreateAPIKeyRequest,
  CreateBasicAuthRequest,
  CreateOAuth2Request,
  CreateServiceAccountRequest,
  CreateCustomCredentialRequest,
} from '@/services/credentialsService';

interface CustomField {
  key: string;
  value: string;
}

interface CredentialFormState {
  name: string;
  description: string;
  provider: string;
  apiKey: string;
  username: string;
  password: string;
  clientId: string;
  clientSecret: string;
  accessToken: string;
  refreshToken: string;
  tokenUrl: string;
  scopes: string;
  jsonKey: string;
  customFields: CustomField[];
}

type CredentialFormAction =
  | { type: 'SET_FIELD'; field: keyof CredentialFormState; value: any }
  | { type: 'RESET' };

const initialState: CredentialFormState = {
  name: '',
  description: '',
  provider: '',
  apiKey: '',
  username: '',
  password: '',
  clientId: '',
  clientSecret: '',
  accessToken: '',
  refreshToken: '',
  tokenUrl: '',
  scopes: '',
  jsonKey: '',
  customFields: [{ key: '', value: '' }],
};

function credentialFormReducer(
  state: CredentialFormState,
  action: CredentialFormAction
): CredentialFormState {
  switch (action.type) {
    case 'SET_FIELD':
      return { ...state, [action.field]: action.value };
    case 'RESET':
      return initialState;
    default:
      return state;
  }
}

export function useCredentialForm() {
  const [state, dispatch] = useReducer(credentialFormReducer, initialState);

  const setField = (field: keyof CredentialFormState, value: any) => {
    dispatch({ type: 'SET_FIELD', field, value });
  };

  const reset = () => {
    dispatch({ type: 'RESET' });
  };

  const buildRequest = (type: CredentialType): any => {
    const baseData = {
      name: state.name.trim(),
      description: state.description.trim(),
      provider: state.provider.trim(),
    };

    switch (type) {
      case 'api_key':
        return {
          ...baseData,
          api_key: state.apiKey,
        } as CreateAPIKeyRequest;

      case 'basic_auth':
        return {
          ...baseData,
          username: state.username,
          password: state.password,
        } as CreateBasicAuthRequest;

      case 'oauth2':
        return {
          ...baseData,
          client_id: state.clientId,
          client_secret: state.clientSecret,
          access_token: state.accessToken || undefined,
          refresh_token: state.refreshToken || undefined,
          token_url: state.tokenUrl || undefined,
          scopes: state.scopes || undefined,
        } as CreateOAuth2Request;

      case 'service_account':
        return {
          ...baseData,
          json_key: state.jsonKey,
        } as CreateServiceAccountRequest;

      case 'custom':
        const customData: Record<string, string> = {};
        state.customFields.forEach((f) => {
          if (f.key.trim() && f.value) {
            customData[f.key.trim()] = f.value;
          }
        });
        return {
          ...baseData,
          data: customData,
        } as CreateCustomCredentialRequest;

      default:
        return baseData;
    }
  };

  const isValid = (type: CredentialType | null): boolean => {
    if (!type || !state.name.trim()) return false;

    switch (type) {
      case 'api_key':
        return !!state.apiKey;
      case 'basic_auth':
        return !!state.username && !!state.password;
      case 'oauth2':
        return !!state.clientId && !!state.clientSecret;
      case 'service_account':
        return !!state.jsonKey;
      case 'custom':
        return state.customFields.some((f) => f.key.trim() && f.value);
      default:
        return false;
    }
  };

  return {
    state,
    setField,
    reset,
    buildRequest,
    isValid,
  };
}
