/**
 * Application Configuration
 * 
 * This file contains the application-wide configuration that can be customized
 * via environment variables. This allows for easy white-labeling and branding.
 */

export const appConfig = {
  /**
   * Application name - displayed throughout the UI
   * Can be customized via VITE_APP_NAME environment variable
   */
  name: import.meta.env.VITE_APP_NAME || 'Whatomate',
  
  /**
   * Application icon path
   * Can be customized via VITE_APP_ICON environment variable
   * Defaults to /favicon.svg
   */
  icon: import.meta.env.VITE_APP_ICON || '/favicon.svg',
  
  /**
   * Application description
   * Can be customized via VITE_APP_DESCRIPTION environment variable
   */
  description: import.meta.env.VITE_APP_DESCRIPTION || 'WhatsApp Business Platform'
}
