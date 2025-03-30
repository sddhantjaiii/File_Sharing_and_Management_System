/// <reference types="react" />
/// <reference types="react-dom" />

import React from 'react';

declare global {
  namespace JSX {
    interface Element extends React.ReactElement<any, any> {}
    interface ElementClass extends React.Component<any> {
      render(): React.ReactNode;
    }
    interface IntrinsicElements {
      div: React.DetailedHTMLProps<React.HTMLAttributes<HTMLDivElement>, HTMLDivElement>;
      main: React.DetailedHTMLProps<React.HTMLAttributes<HTMLElement>, HTMLElement>;
      span: React.DetailedHTMLProps<React.HTMLAttributes<HTMLSpanElement>, HTMLSpanElement>;
      button: React.DetailedHTMLProps<React.ButtonHTMLAttributes<HTMLButtonElement>, HTMLButtonElement>;
      input: React.DetailedHTMLProps<React.InputHTMLAttributes<HTMLInputElement>, HTMLInputElement>;
      form: React.DetailedHTMLProps<React.FormHTMLAttributes<HTMLFormElement>, HTMLFormElement>;
      label: React.DetailedHTMLProps<React.LabelHTMLAttributes<HTMLLabelElement>, HTMLLabelElement>;
    }
  }
}

declare module 'react-hot-toast' {
  export const Toaster: React.FC<{
    position?: 'top-left' | 'top-right' | 'top-center' | 'bottom-left' | 'bottom-right' | 'bottom-center';
  }>;
  export function toast(message: string, options?: any): void;
}

interface User {
  id: number;
  username: string;
  email: string;
}

interface File {
  id: number;
  filename: string;
  originalName: string;
  size: number;
  mimeType: string;
  isPublic: boolean;
  expiresAt?: string;
  createdAt: string;
  updatedAt: string;
}

interface LoginResponse {
  token: string;
  user: User;
}

interface ApiError {
  error: string;
}

declare module '*.tsx' {
  const content: React.FC;
  export default content;
} 