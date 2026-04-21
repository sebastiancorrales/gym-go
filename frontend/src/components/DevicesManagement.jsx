import { useState, useEffect } from 'react';
import api from '../utils/api';
import Modal from './Modal';
import Toast from './Toast';
import ConfirmDialog from './ConfirmDialog';

const DEFAULT_BAUD = 9600;

const emptyForm = () => ({ name: '', location: '', com_port: '', baud_rate: DEFAULT_BAUD, notes: '' });

export default function DevicesManagement() {
  const [devices, setDevices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState(null);
  const [form, setForm] = useState(emptyForm());
  const [saving, setSaving] = useState(false);
  const [toast, setToast] = useState(null);
  const [confirmDialog, setConfirmDialog] = useState({ open: false, onConfirm: null });
  const [triggering, setTriggering] = useState({});

  useEffect(() => { fetchDevices(); }, []);

  const fetchDevices = async () => {
    setLoading(true);
    try {
      const res = await api.get('/devices');
      if (res.ok) setDevices(await res.json() || []);
    } catch {}
    finally { setLoading(false); }
  };

  const openCreate = () => {
    setEditing(null);
    setForm(emptyForm());
    setShowModal(true);
  };

  const openEdit = (device) => {
    setEditing(device);
    setForm({
      name: device.name,
      location: device.location || '',
      com_port: device.com_port || '',
      baud_rate: device.baud_rate || DEFAULT_BAUD,
      notes: device.notes || '',
    });
    setShowModal(true);
  };

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      const body = { ...form, baud_rate: Number(form.baud_rate) };
      const res = editing
        ? await api.put(`/devices/${editing.id}`, body)
        : await api.post('/devices', body);

      if (res.ok) {
        setToast({ message: editing ? 'Dispositivo actualizado' : 'Dispositivo creado', type: 'success' });
        setShowModal(false);
        fetchDevices();
      } else {
        const err = await res.json();
        setToast({ message: err.error || 'Error al guardar', type: 'error' });
      }
    } catch {
      setToast({ message: 'Error de conexión', type: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = (device) => {
    setConfirmDialog({
      open: true,
      onConfirm: async () => {
        const res = await api.delete(`/devices/${device.id}`);
        if (res.ok) {
          setToast({ message: 'Dispositivo eliminado', type: 'success' });
          fetchDevices();
        } else {
          setToast({ message: 'Error al eliminar', type: 'error' });
        }
      },
    });
  };

  const handleTrigger = async (device) => {
    setTriggering(prev => ({ ...prev, [device.id]: true }));
    try {
      const res = await api.post(`/devices/${device.id}/trigger`, {});
      if (res.ok) {
        setToast({ message: `Relé activado: ${device.name}`, type: 'success' });
      } else {
        const err = await res.json();
        setToast({ message: err.error || 'Error al activar relé', type: 'error' });
      }
    } catch {
      setToast({ message: 'Error de conexión', type: 'error' });
    } finally {
      setTriggering(prev => ({ ...prev, [device.id]: false }));
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">Dispositivos Externos</h2>
          <p className="text-gray-600 mt-1">Gestiona relés Arduino para control de acceso</p>
        </div>
        <button
          onClick={openCreate}
          className="px-4 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-lg hover:from-emerald-600 hover:to-cyan-600 transition font-medium"
        >
          + Nuevo Dispositivo
        </button>
      </div>

      <div className="bg-white rounded-lg shadow-md overflow-hidden">
        {loading ? (
          <div className="p-8 text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-emerald-600 mx-auto" />
            <p className="text-gray-600 mt-4">Cargando dispositivos...</p>
          </div>
        ) : devices.length === 0 ? (
          <div className="p-12 text-center text-gray-500">
            <svg className="mx-auto w-12 h-12 text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18" />
            </svg>
            <p className="font-medium">No hay dispositivos registrados</p>
            <p className="text-sm mt-1">Agrega un Arduino con relé para controlar el acceso</p>
          </div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                {['Nombre', 'Ubicación', 'Puerto COM', 'Baud Rate', 'Estado', 'Acciones'].map(h => (
                  <th key={h} className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {devices.map(device => (
                <tr key={device.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="font-medium text-gray-900">{device.name}</div>
                    {device.notes && <div className="text-xs text-gray-400">{device.notes}</div>}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{device.location || '—'}</td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="font-mono text-sm text-gray-900">{device.com_port || '—'}</span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{device.baud_rate || DEFAULT_BAUD}</td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 text-xs font-semibold rounded-full ${device.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-600'}`}>
                      {device.is_active ? 'Activo' : 'Inactivo'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm flex gap-3">
                    <button
                      onClick={() => handleTrigger(device)}
                      disabled={!device.is_active || !device.com_port || triggering[device.id]}
                      className="text-emerald-600 hover:text-emerald-900 font-medium disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                      {triggering[device.id] ? 'Abriendo...' : 'Abrir'}
                    </button>
                    <button onClick={() => openEdit(device)} className="text-blue-600 hover:text-blue-900 font-medium">Editar</button>
                    <button onClick={() => handleDelete(device)} className="text-red-600 hover:text-red-900 font-medium">Eliminar</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <Modal
        isOpen={showModal}
        onClose={() => setShowModal(false)}
        title={editing ? 'Editar Dispositivo' : 'Nuevo Dispositivo'}
        maxWidth="lg"
      >
        <form onSubmit={handleSave} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Nombre *</label>
            <input
              type="text"
              value={form.name}
              onChange={e => setForm({ ...form, name: e.target.value })}
              required
              placeholder="Ej: Torniquete Principal"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Ubicación</label>
            <input
              type="text"
              value={form.location}
              onChange={e => setForm({ ...form, location: e.target.value })}
              placeholder="Ej: Entrada principal"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Puerto COM</label>
              <input
                type="text"
                value={form.com_port}
                onChange={e => setForm({ ...form, com_port: e.target.value })}
                placeholder="Ej: COM5"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent font-mono"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Baud Rate</label>
              <select
                value={form.baud_rate}
                onChange={e => setForm({ ...form, baud_rate: Number(e.target.value) })}
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
              >
                {[9600, 19200, 38400, 57600, 115200].map(b => (
                  <option key={b} value={b}>{b}</option>
                ))}
              </select>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Notas</label>
            <input
              type="text"
              value={form.notes}
              onChange={e => setForm({ ...form, notes: e.target.value })}
              placeholder="Descripción opcional"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
          </div>
          <div className="flex gap-3 pt-2">
            <button type="button" onClick={() => setShowModal(false)} className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-xl hover:bg-gray-50 transition">
              Cancelar
            </button>
            <button type="submit" disabled={saving} className="flex-1 px-4 py-2 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-xl hover:from-emerald-600 hover:to-cyan-600 transition disabled:opacity-50">
              {saving ? 'Guardando...' : 'Guardar'}
            </button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        isOpen={confirmDialog.open}
        onClose={() => setConfirmDialog({ open: false, onConfirm: null })}
        onConfirm={confirmDialog.onConfirm}
        title="Eliminar dispositivo"
        message="¿Está seguro de eliminar este dispositivo? Esta acción no se puede deshacer."
        confirmText="Eliminar"
        variant="danger"
      />

      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  );
}
