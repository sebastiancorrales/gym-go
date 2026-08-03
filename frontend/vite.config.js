import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react({
      babel: {
        plugins: [['babel-plugin-react-compiler']],
      },
    }),
  ],
  base: '/',
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    chunkSizeWarningLimit: 700,
    rollupOptions: {
      output: {
        // Solo React va en un chunk manual, y es por cache del navegador entre
        // despliegues: no cambia, asi que no hay que volver a descargarlo cada vez
        // que se toca el codigo de la aplicacion.
        //
        // recharts, xlsx y jspdf NO se listan aqui a proposito. Al ponerlos, Vite
        // los trataba como dependencias estaticas del entry y les añadia un
        // <link rel="modulepreload"> en index.html, con lo que el navegador los
        // descargaba en el primer paint igualmente y la carga diferida no servia de
        // nada. Dejando que los import() dinamicos generen sus propios chunks, solo
        // se descargan cuando de verdad se necesitan.
        //
        // La forma de funcion (en vez de un objeto) es necesaria para que react
        // -vendor recoja tambien react/jsx-runtime: con ['react','react-dom'] el
        // chunk salia de 0 bytes y React terminaba en el chunk principal.
        manualChunks(id) {
          if (
            id.includes('node_modules/react/') ||
            id.includes('node_modules/react-dom/') ||
            id.includes('node_modules/scheduler/')
          ) {
            return 'react-vendor';
          }
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
