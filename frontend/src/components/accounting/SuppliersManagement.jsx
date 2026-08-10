import { useState, useEffect } from 'react';
import api from '../../utils/api';
import Modal from '../Modal';
import Toast from '../Toast';
import ConfirmDialog from '../ConfirmDialog';
import SkeletonTable from '../SkeletonTable';
import EmptyState from '../EmptyState';

// Proveedores: a quién se le compra.
//
// Desactivar en lugar de borrar. Un proveedor referenciado por un gasto no puede
// desaparecer sin dejar el histórico contable roto, y el proyecto no tiene soft
// delete.

const EMPTY_FORM = {
  name: '', tax_id: '', contact_name: '',
  phone: '', email: '', address: '', notes: '', status: 'active',
};

export default function SuppliersManagement() {
  const [suppliers, setSuppliers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [showInactive, setShowInactive] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState(null);
  const [form, setForm] = useState(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [toast, setToast] = useState(null);
  const [confirm, setConfirm] = useState({ open: false, onConfirm: null, message: '' });

  const fetchSuppliers = async () => {
    setLoading(true);
    try {
      const res = await api.get('/suppliers', { noCache: true });
      setSuppliers(res.ok ? (await res.json()) || [] : []);
    } catch {
      setToast({ message: 'Error al cargar proveedores', type: 'error' });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchSuppliers(); }, []);

  const openCreate = () => { setEditing(null); setForm(EMPTY_FORM); setShowModal(true); };

  const openEdit = (supplier) => {
    setEditing(supplier);
    setForm({
      name: supplier.name || '',
      tax_id: supplier.tax_id || '',
      contact_name: supplier.contact_name || '',
      phone: supplier.phone || '',
      email: supplier.email || '',
      address: supplier.address || '',
      notes: supplier.notes || '',
      status: supplier.status || 'active',
    });
    setShowModal(true);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (saving) return;
    setSaving(true);
    try {
      const res = editing
        ? await api.put(`/suppliers/${editing.id}`, form)
        : await api.post('/suppliers', form);

      if (res.ok) {
        setToast({ message: editing ? 'Proveedor actualizado' : 'Proveedor creado', type: 'success' });
        setShowModal(false);
        fetchSuppliers();
      } else {
        // El caso concreto: NIT repetido dentro del mismo gimnasio.
        const body = await res.json().catch(() => ({}));
        setToast({ message: body.message || 'No se pudo guardar el proveedor', type: 'error' });
      }
    } catch {
      setToast({ message: 'Error al guardar el proveedor', type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDeactivate = (supplier) => {
    setConfirm({
      open: true,
      message: `${supplier.name} dejará de aparecer al registrar gastos. Los gastos ya registrados a su nombre se conservan.`,
      onConfirm: async () => {
        try {
          const res = await api.delete(`/suppliers/${supplier.id}`);
          setToast(res.ok
            ? { message: 'Proveedor desactivado', type: 'success' }
            : { message: 'No se pudo desactivar', type: 'error' });
          if (res.ok) fetchSuppliers();
        } catch {
          setToast({ message: 'Error al desactivar', type: 'error' });
        } finally {
          setConfirm({ open: false, onConfirm: null, message: '' });
        }
      },
    });
  };

  const visible = suppliers
    .filter(s => showInactive || s.status === 'active')
    .filter(s => {
      const q = search.trim().toLowerCase();
      if (!q) return true;
      return [s.name, s.tax_id, s.contact_name, s.phone].some(v => (v || '').toLowerCase().includes(q));
    });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">Proveedores</h2>
          <p className="text-sm text-gray-500 mt-0.5">A quién se le compra la mercancía y los servicios.</p>
        </div>
        <button
          onClick={openCreate}
          className="px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition font-semibold text-sm"
        >
          + Nuevo Proveedor
        </button>
      </div>

      <div className="bg-white rounded-xl border border-gray-100 shadow-sm p-4 flex flex-wrap items-center gap-3">
        <input
          type="text"
          value={search}
          onChange={e => setSearch(e.target.value)}
          placeholder="Buscar por nombre, NIT o contacto..."
          className="flex-1 min-w-[220px] px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
        />
        <label className="flex items-center gap-2 text-sm text-gray-600">
          <input type="checkbox" checked={showInactive} onChange={e => setShowInactive(e.target.checked)} />
          Mostrar inactivos
        </label>
      </div>

      {loading ? (
        <SkeletonTable cols={5} rows={6} />
      ) : visible.length === 0 ? (
        <EmptyState
          icon="list"
          title="Sin proveedores"
          description="Registra a quién le compras para poder asociarlo a tus gastos."
        />
      ) : (
        <div className="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-[11px] font-semibold text-gray-500 uppercase tracking-wide bg-gray-50">
                  <th className="px-6 py-3">Proveedor</th>
                  <th className="px-6 py-3">NIT</th>
                  <th className="px-6 py-3">Contacto</th>
                  <th className="px-6 py-3">Estado</th>
                  <th className="px-6 py-3 text-right">Acciones</th>
                </tr>
              </thead>
              <tbody>
                {visible.map(s => (
                  <tr key={s.id} className="border-t border-gray-50 hover:bg-gray-50">
                    <td className="px-6 py-3">
                      <p className="font-semibold text-gray-900">{s.name}</p>
                      {s.address && <p className="text-xs text-gray-400">{s.address}</p>}
                    </td>
                    <td className="px-6 py-3 text-gray-600">{s.tax_id || '—'}</td>
                    <td className="px-6 py-3 text-gray-600">
                      {s.contact_name || '—'}
                      {s.phone && <span className="block text-xs text-gray-400">{s.phone}</span>}
                    </td>
                    <td className="px-6 py-3">
                      <span className={`px-2 py-1 text-xs font-semibold rounded-full ${
                        s.status === 'active'
                          ? 'bg-[#DCFCE7] text-[#065F46]'
                          : 'bg-gray-100 text-gray-600'
                      }`}>
                        {s.status === 'active' ? 'Activo' : 'Inactivo'}
                      </span>
                    </td>
                    <td className="px-6 py-3 text-right whitespace-nowrap">
                      <button onClick={() => openEdit(s)}
                        className="text-[#1272D6] hover:text-[#0D5BAD] font-medium mr-4">
                        Editar
                      </button>
                      {s.status === 'active' && (
                        <button onClick={() => handleDeactivate(s)}
                          className="text-red-600 hover:text-red-700 font-medium">
                          Desactivar
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={editing ? 'Editar Proveedor' : 'Nuevo Proveedor'}
        maxWidth="md"
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Nombre *</label>
            <input type="text" required value={form.name}
              onChange={e => setForm({ ...form, name: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">NIT</label>
              <input type="text" value={form.tax_id}
                onChange={e => setForm({ ...form, tax_id: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
              <p className="mt-1 text-xs text-gray-400">Opcional, pero no puede repetirse.</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Contacto</label>
              <input type="text" value={form.contact_name}
                onChange={e => setForm({ ...form, contact_name: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Teléfono</label>
              <input type="tel" value={form.phone}
                onChange={e => setForm({ ...form, phone: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
              <input type="email" value={form.email}
                onChange={e => setForm({ ...form, email: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Dirección</label>
            <input type="text" value={form.address}
              onChange={e => setForm({ ...form, address: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Notas</label>
            <textarea rows={2} value={form.notes}
              onChange={e => setForm({ ...form, notes: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent" />
          </div>

          {editing && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Estado</label>
              <select value={form.status}
                onChange={e => setForm({ ...form, status: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent">
                <option value="active">Activo</option>
                <option value="inactive">Inactivo</option>
              </select>
            </div>
          )}

          <div className="flex gap-3 pt-2">
            <button type="button" onClick={() => setShowModal(false)}
              className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition">
              Cancelar
            </button>
            <button type="submit" disabled={saving}
              className="flex-1 px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition disabled:opacity-50">
              {saving ? 'Guardando...' : (editing ? 'Actualizar' : 'Crear')}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        isOpen={confirm.open}
        onClose={() => setConfirm({ open: false, onConfirm: null, message: '' })}
        onConfirm={confirm.onConfirm}
        title="Desactivar proveedor"
        message={confirm.message}
        confirmText="Desactivar"
      />

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  );
}
