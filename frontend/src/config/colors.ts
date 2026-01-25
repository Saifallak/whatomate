/**
 * Color Configuration System
 * 
 * This file handles dynamic color theming for the application.
 * Colors can be customized via environment variables in HEX format.
 * The system automatically:
 * - Converts HEX to HSL for CSS variables
 * - Generates gradient variations
 * - Calculates lighter/darker shades
 * - Supports both dark and light modes
 */

interface HSL {
    h: number
    s: number
    l: number
}

interface ColorConfig {
    primary: string
    primaryGradientStart: string
    primaryGradientEnd: string
    secondary?: string
    accent?: string
}

/**
 * Convert HEX color to HSL
 * @param hex - Color in HEX format (e.g., "#10b981")
 * @returns HSL object with h (0-360), s (0-100), l (0-100)
 */
export function hexToHSL(hex: string): HSL {
    // Remove # if present
    hex = hex.replace('#', '')

    // Convert to RGB
    const r = parseInt(hex.substring(0, 2), 16) / 255
    const g = parseInt(hex.substring(2, 4), 16) / 255
    const b = parseInt(hex.substring(4, 6), 16) / 255

    const max = Math.max(r, g, b)
    const min = Math.min(r, g, b)
    let h = 0
    let s = 0
    const l = (max + min) / 2

    if (max !== min) {
        const d = max - min
        s = l > 0.5 ? d / (2 - max - min) : d / (max + min)

        switch (max) {
            case r:
                h = ((g - b) / d + (g < b ? 6 : 0)) / 6
                break
            case g:
                h = ((b - r) / d + 2) / 6
                break
            case b:
                h = ((r - g) / d + 4) / 6
                break
        }
    }

    return {
        h: Math.round(h * 360),
        s: Math.round(s * 100),
        l: Math.round(l * 100)
    }
}

/**
 * Adjust lightness of an HSL color
 * @param hsl - HSL color object
 * @param adjustment - Amount to adjust lightness (-100 to +100)
 * @returns New HSL object with adjusted lightness
 */
export function adjustLightness(hsl: HSL, adjustment: number): HSL {
    return {
        h: hsl.h,
        s: hsl.s,
        l: Math.max(0, Math.min(100, hsl.l + adjustment))
    }
}

/**
 * Adjust saturation of an HSL color
 * @param hsl - HSL color object
 * @param adjustment - Amount to adjust saturation (-100 to +100)
 * @returns New HSL object with adjusted saturation
 */
export function adjustSaturation(hsl: HSL, adjustment: number): HSL {
    return {
        h: hsl.h,
        s: Math.max(0, Math.min(100, hsl.s + adjustment)),
        l: hsl.l
    }
}

/**
 * Generate gradient colors from a base color
 * Creates a lighter start and darker end for gradients
 * @param baseHex - Base color in HEX format
 * @returns Object with start and end gradient colors in HSL format
 */
export function generateGradient(baseHex: string): { start: HSL; end: HSL } {
    const base = hexToHSL(baseHex)

    // For gradient:
    // - Start: slightly lighter and more saturated
    // - End: slightly darker
    const start = adjustSaturation(adjustLightness(base, 8), 5)
    const end = adjustLightness(base, -8)

    return { start, end }
}

/**
 * Format HSL object to CSS HSL string
 * @param hsl - HSL color object
 * @returns CSS HSL string (e.g., "160 84% 39%")
 */
export function hslToString(hsl: HSL): string {
    return `${hsl.h} ${hsl.s}% ${hsl.l}%`
}

/**
 * Get color configuration from environment variables
 * Falls back to default WhatsApp-style green if not specified
 */
export function getColorConfig(): ColorConfig {
    const primaryHex = import.meta.env.VITE_PRIMARY_COLOR || '#10b981'
    const secondaryHex = import.meta.env.VITE_SECONDARY_COLOR
    const accentHex = import.meta.env.VITE_ACCENT_COLOR

    console.log('🎨 [Color Config] Reading environment variables...')
    console.log('  VITE_PRIMARY_COLOR:', import.meta.env.VITE_PRIMARY_COLOR)
    console.log('  Resolved primaryHex:', primaryHex)

    // Check if custom gradient colors are provided
    const gradientStartHex = import.meta.env.VITE_PRIMARY_GRADIENT_START
    const gradientEndHex = import.meta.env.VITE_PRIMARY_GRADIENT_END

    let gradientStart: string
    let gradientEnd: string

    if (gradientStartHex && gradientEndHex) {
        // Use custom gradient colors
        console.log('  Using custom gradient colors')
        gradientStart = hslToString(hexToHSL(gradientStartHex))
        gradientEnd = hslToString(hexToHSL(gradientEndHex))
    } else {
        // Auto-generate gradient from primary color
        console.log('  Auto-generating gradient...')
        const gradient = generateGradient(primaryHex)
        gradientStart = hslToString(gradient.start)
        gradientEnd = hslToString(gradient.end)
        console.log('  Gradient start:', gradientStart)
        console.log('  Gradient end:', gradientEnd)
    }

    const config = {
        primary: hslToString(hexToHSL(primaryHex)),
        primaryGradientStart: gradientStart,
        primaryGradientEnd: gradientEnd,
        secondary: secondaryHex ? hslToString(hexToHSL(secondaryHex)) : undefined,
        accent: accentHex ? hslToString(hexToHSL(accentHex)) : undefined
    }

    console.log('  Final config:', config)
    return config
}

/**
 * Apply color configuration to CSS variables
 * This function should be called on app initialization
 */
export function applyColorTheme(): void {
    console.log('🎨 [Apply Theme] Starting...')
    const colors = getColorConfig()
    const root = document.documentElement

    // Apply primary color
    console.log('  Setting CSS variables...')
    root.style.setProperty('--primary', colors.primary)
    root.style.setProperty('--ring', colors.primary)

    // Apply gradient colors
    root.style.setProperty('--primary-gradient-start', colors.primaryGradientStart)
    root.style.setProperty('--primary-gradient-end', colors.primaryGradientEnd)

    console.log('  --primary:', colors.primary)
    console.log('  --primary-gradient-start:', colors.primaryGradientStart)
    console.log('  --primary-gradient-end:', colors.primaryGradientEnd)

    // Apply secondary color if provided
    if (colors.secondary) {
        root.style.setProperty('--secondary-custom', colors.secondary)
    }

    // Apply accent color if provided
    if (colors.accent) {
        root.style.setProperty('--accent-custom', colors.accent)
    }

    // Update gradient background for body
    const primaryHSL = hexToHSL(import.meta.env.VITE_PRIMARY_COLOR || '#10b981')
    const gradientColor = `rgba(${hslToRGB(primaryHSL).join(', ')}, 0.08)`
    root.style.setProperty('--gradient-color', gradientColor)

    console.log('✅ [Apply Theme] Complete!')
    console.log('💡 Check in DevTools: getComputedStyle(document.documentElement).getPropertyValue("--primary")')
}

/**
 * Convert HSL to RGB
 * @param hsl - HSL color object
 * @returns RGB array [r, g, b] with values 0-255
 */
function hslToRGB(hsl: HSL): [number, number, number] {
    const h = hsl.h / 360
    const s = hsl.s / 100
    const l = hsl.l / 100

    let r: number, g: number, b: number

    if (s === 0) {
        r = g = b = l
    } else {
        const hue2rgb = (p: number, q: number, t: number) => {
            if (t < 0) t += 1
            if (t > 1) t -= 1
            if (t < 1 / 6) return p + (q - p) * 6 * t
            if (t < 1 / 2) return q
            if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6
            return p
        }

        const q = l < 0.5 ? l * (1 + s) : l + s - l * s
        const p = 2 * l - q

        r = hue2rgb(p, q, h + 1 / 3)
        g = hue2rgb(p, q, h)
        b = hue2rgb(p, q, h - 1 / 3)
    }

    return [
        Math.round(r * 255),
        Math.round(g * 255),
        Math.round(b * 255)
    ]
}

/**
 * Preset color schemes for quick setup
 */
export const colorPresets = {
    whatsapp: '#10b981',      // Emerald green (default)
    telegram: '#0088cc',      // Telegram blue
    messenger: '#0084ff',     // Facebook Messenger blue
    viber: '#7360f2',         // Viber purple
    signal: '#3a76f0',        // Signal blue
    discord: '#5865f2',       // Discord blurple
    slack: '#4a154b',         // Slack aubergine
    teams: '#6264a7',         // Microsoft Teams purple
    custom: '#10b981'         // Custom (same as default)
}
