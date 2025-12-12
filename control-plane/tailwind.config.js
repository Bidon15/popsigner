/** @type {import('tailwindcss').Config} */
// POPSigner 1980s CRT Terminal / Bloomberg Aesthetic
// Amber + Green phosphor on black. Stranger Things S5 vibes.
module.exports = {
  content: [
    "./templates/**/*.templ",
    "./templates/**/*_templ.go",
    "./static/js/**/*.js",
  ],
  darkMode: 'class', // Always dark (CRTs were black)
  theme: {
    extend: {
      colors: {
        // PRIMARY: Amber Phosphor (CRT glow)
        phosphor: {
          amber: '#FFB000',           // Main amber
          'amber-bright': '#FFCC00',  // Highlighted
          'amber-dim': '#CC8800',     // Dimmed
          'amber-dark': '#333300',    // Borders
        },
        // SECONDARY: Green Phosphor
        crt: {
          green: '#33FF00',           // Bright green
          'green-dim': '#228B22',     // Forest green
          'green-dark': '#1A4D1A',    // Very dark green
          'green-bg': '#0D1A0D',      // Subtle green tint
        },
        // ALERT: Terminal Red
        terminal: {
          red: '#FF3333',
          'red-dim': '#CC2222',
          black: '#000000',
          // Backgrounds - CRT black
          'bg': '#000000',
          'card': '#0A0A0A',
          'elevated': '#111111',
          'border': '#333300',
          'border-highlight': '#FFB000',
          // Text
          'text': '#FFB000',
          'muted': '#CC8800',
          'dim': '#666600',
          // Accent
          'accent': '#FFB000',
          'accent-hover': '#FFCC00',
          'teal': '#33FF00',
          'teal-dim': '#228B22',
        },
      },
      fontFamily: {
        // Monospace ONLY for terminal feel
        mono: ['IBM Plex Mono', 'VT323', 'Courier New', 'monospace'],
        terminal: ['VT323', 'IBM Plex Mono', 'monospace'],
        display: ['IBM Plex Mono', 'monospace'],
        body: ['IBM Plex Mono', 'monospace'],
      },
      fontSize: {
        base: ['1rem', { lineHeight: '1.6' }],
      },
      borderRadius: {
        'xl': '0.25rem',
        '2xl': '0.375rem',
        '3xl': '0.5rem',
      },
      boxShadow: {
        // Phosphor glow effects
        'glow-amber': '0 0 10px #FFB000, 0 0 20px #FFB000',
        'glow-green': '0 0 10px #33FF00, 0 0 20px #33FF00',
        'glow-red': '0 0 10px #FF3333',
        'glow-amber-sm': '0 0 5px #FFB000',
        'glow-green-sm': '0 0 5px #33FF00',
      },
      animation: {
        'fade-in': 'fadeIn 0.5s ease-out forwards',
        'slide-up': 'slideUp 0.5s ease-out forwards',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'pulse-soft': 'pulseSoft 3s ease-in-out infinite',
        'blink': 'blink 1s step-end infinite',
        'scanline': 'scanline 8s linear infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(20px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(10px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        pulseSoft: {
          '0%, 100%': { opacity: '0.5' },
          '50%': { opacity: '1' },
        },
        blink: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0' },
        },
        scanline: {
          '0%': { transform: 'translateY(-100%)' },
          '100%': { transform: 'translateY(100vh)' },
        },
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'crt-grid': 'linear-gradient(to right, rgba(51, 51, 0, 0.3) 1px, transparent 1px), linear-gradient(to bottom, rgba(51, 51, 0, 0.3) 1px, transparent 1px)',
      },
    },
  },
  plugins: [],
}
