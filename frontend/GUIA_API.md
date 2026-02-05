# Guía de Integración de API con Manejo Automático de Token

## ✅ Ya implementado

He creado un sistema que **automáticamente redirige al login** cuando el token expira.

### Archivo: `src/utils/api.js`

Este archivo contiene:
- ✅ Manejo automático de tokens de autorización
- ✅ Detección de tokens expirados (error 401)
- ✅ Redirección automática al login
- ✅ Limpieza de localStorage

## 📝 Cómo usar en tus componentes

### Antes (con fetch manual):
```jsx
const response = await fetch('http://localhost:8080/api/v1/users', {
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('access_token')}`
  }
});
```

### Después (con api automático):
```jsx
import api from '../utils/api';

const response = await api.get('/users');
```

## 🔧 Métodos disponibles

```javascript
import api from '../utils/api';

// GET
const response = await api.get('/users');
const response = await api.get('/users/123');

// POST
const response = await api.post('/users', {
  email: 'user@example.com',
  first_name: 'Juan'
});

// PUT
const response = await api.put('/users/123', {
  first_name: 'Juan Actualizado'
});

// DELETE
const response = await api.delete('/users/123');
```

## 📦 Ejemplo completo

```jsx
import { useState, useEffect } from 'react';
import api from '../utils/api';

export default function MiComponente() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);

  // GET - Listar datos
  const fetchData = async () => {
    setLoading(true);
    try {
      const response = await api.get('/users');
      
      if (response.ok) {
        const data = await response.json();
        setData(data.data || []);
      }
    } catch (error) {
      console.error('Error:', error);
      alert(error.message); // Mostrará "Sesión expirada..." si el token expira
    } finally {
      setLoading(false);
    }
  };

  // POST - Crear nuevo
  const createUser = async (userData) => {
    try {
      const response = await api.post('/users', userData);
      
      if (response.ok) {
        const result = await response.json();
        alert('Usuario creado exitosamente');
        fetchData(); // Recargar lista
      } else {
        const error = await response.json();
        alert(error.error || 'Error al crear usuario');
      }
    } catch (error) {
      alert(error.message);
    }
  };

  // PUT - Actualizar
  const updateUser = async (userId, userData) => {
    try {
      const response = await api.put(`/users/${userId}`, userData);
      
      if (response.ok) {
        alert('Usuario actualizado');
        fetchData();
      }
    } catch (error) {
      alert(error.message);
    }
  };

  // DELETE - Eliminar
  const deleteUser = async (userId) => {
    try {
      const response = await api.delete(`/users/${userId}`);
      
      if (response.ok) {
        alert('Usuario eliminado');
        fetchData();
      }
    } catch (error) {
      alert(error.message);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  return (
    <div>
      {/* Tu UI aquí */}
    </div>
  );
}
```

## 🎯 Componentes que debes actualizar

Ya actualicé `UsersManagement.jsx` como ejemplo. Aplica el mismo patrón a:

1. ✅ `UsersManagement.jsx` - Ya actualizado
2. ⏳ `SubscriptionsManagement.jsx`
3. ⏳ `PlansManagement.jsx`
4. ⏳ `MembersTab.jsx`
5. ⏳ `ClassesTab.jsx`
6. ⏳ `AttendanceTab.jsx`
7. ⏳ `AccessManagement.jsx`
8. ⏳ `CheckIn.jsx`
9. ⏳ `Dashboard.jsx`

## 🚀 Pasos para actualizar un componente

1. **Importar la utilidad:**
   ```jsx
   import api from '../utils/api';
   ```

2. **Reemplazar fetch por api:**
   - `fetch('http://localhost:8080/api/v1/users')` → `api.get('/users')`
   - Remover manualmente el header de `Authorization`
   - Remover `localStorage.getItem('access_token')`

3. **Agregar manejo de errores:**
   ```jsx
   try {
     const response = await api.get('/endpoint');
     // ...
   } catch (error) {
     alert(error.message); // Muestra el mensaje de sesión expirada
   }
   ```

## ⚡ Beneficios

- ✅ **No más código duplicado** para manejar tokens
- ✅ **Redirección automática** cuando expira el token
- ✅ **Mensajes de error claros** para el usuario
- ✅ **Código más limpio y mantenible**
- ✅ **Manejo centralizado** de autenticación

## 🔐 Qué pasa cuando el token expira

1. Usuario hace una petición
2. Backend responde con 401 Unauthorized
3. `api.js` detecta el error automáticamente
4. Limpia el localStorage (tokens y usuario)
5. Redirige a `/` (página de login)
6. Usuario ve el mensaje: "Sesión expirada. Por favor, inicia sesión nuevamente."

¡No necesitas hacer nada más! 🎉
