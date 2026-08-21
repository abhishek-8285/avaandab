declare module 'expo-image-picker' {
  export const MediaTypeOptions: any;
  export function requestMediaLibraryPermissionsAsync(): Promise<any>;
  export function requestCameraPermissionsAsync(): Promise<any>;
  export function launchImageLibraryAsync(options?: any): Promise<any>;
  export function launchCameraAsync(options?: any): Promise<any>;
}
declare module 'react-native-signature-canvas' {
  const SignatureCanvas: any;
  export default SignatureCanvas;
}
declare module 'react-native-webview' {
  export const WebView: any;
}
declare module 'i18next' {
  const i18next: any;
  export default i18next;
}
declare module 'react-i18next' {
  export const initReactI18next: any;
  export function useTranslation(): any;
}
declare module '*.json' {
  const value: any;
  export default value;
}
