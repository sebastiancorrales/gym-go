import { useState, useEffect } from 'react';
import api from '../utils/api';
import Modal from './Modal';
import Toast from './Toast';
import ConfirmDialog from './ConfirmDialog';

const DEFAULT_BAUD = 9600;
const DEVICE_TYPE_RELAY = 'RELAY';
const DEVICE_TYPE_HIKVISION = 'HIKVISION_TERMINAL';

const emptyForm = () => ({
  name: '',
  device_type: DEVICE_TYPE_RELAY,
  location: '',
  com_port: '',
  baud_rate: DEFAULT_BAUD,
  ip_address: '',
  http_port: 80,
  username: 'admin',
  password: '',
  door_no: 1,
  notes: '',
});

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
    let hikvision = {};
    try { hikvision = device.configuration ? JSON.parse(device.configuration) : {}; } catch { /* ignore malformed config */ }
    setForm({
      name: device.name,
      device_type: device.device_type || DEVICE_TYPE_RELAY,
      location: device.location || '',
      com_port: device.com_port || '',
      baud_rate: device.baud_rate || DEFAULT_BAUD,
      ip_address: device.ip_address || '',
      http_port: hikvision.http_port || 80,
      username: hikvision.username || 'admin',
      // The backend never sends the stored password back (see redactDeviceSecret
      // in device_handler.go) — leave blank to keep the current one.
      password: '',
      door_no: hikvision.door_no || 1,
      notes: device.notes || '',
    });
    setShowModal(true);
  };

  const handleSave = async (e) => {
    e.preventDefault();
    setSaving(true);
    try {
      const isHikvision = form.device_type === DEVICE_TYPE_HIKVISION;
      const body = {
        name: form.name,
        device_type: form.device_type,
        location: form.location,
        notes: form.notes,
        ...(isHikvision
          ? {
              ip_address: form.ip_address,
              hikvision: {
                http_port: Number(form.http_port) || 80,
                username: form.username,
                password: form.password, // blank = keep the existing one (merged server-side)
                door_no: Number(form.door_no) || 1,
              },
            }
          : {
              com_port: form.com_port,
              baud_rate: Number(form.baud_rate),
            }),
      };
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

  const canTrigger = (device) =>
    device.device_type === DEVICE_TYPE_HIKVISION ? !!device.ip_address : !!device.com_port;

  const handleTrigger = async (device) => {
    setTriggering(prev => ({ ...prev, [device.id]: true }));
    try {
      const res = await api.post(`/devices/${device.id}/trigger`, {});
      if (res.ok) {
        setToast({ message: `Puerta abierta: ${device.name}`, type: 'success' });
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
          <p className="text-gray-600 mt-1">Gestiona relés Arduino y terminales Hikvision para control de acceso</p>
        </div>
        <button
          onClick={openCreate}
          className="px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition font-medium"
        >
          + Nuevo Dispositivo
        </button>
      </div>

      <div className="bg-white rounded-lg shadow-md overflow-hidden">
        {loading ? (
          <div className="p-8 text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-[#1272D6] mx-auto" />
            <p className="text-gray-600 mt-4">Cargando dispositivos...</p>
          </div>
        ) : devices.length === 0 ? (
          <div className="p-12 text-center text-gray-500">
            <svg className="mx-auto w-12 h-12 text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18" />
            </svg>
            <p className="font-medium">No hay dispositivos registrados</p>
            <p className="text-sm mt-1">Agrega un relé Arduino o un terminal Hikvision para controlar el acceso</p>
          </div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                {['Nombre', 'Tipo', 'Ubicación', 'Conexión', 'Estado', 'Acciones'].map(h => (
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
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                    {device.device_type === DEVICE_TYPE_HIKVISION ? 'Terminal Hikvision' : 'Relé (serial)'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">{device.location || '—'}</td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="font-mono text-sm text-gray-900">
                      {device.device_type === DEVICE_TYPE_HIKVISION
                        ? (device.ip_address || '—')
                        : `${device.com_port || '—'} @ ${device.baud_rate || DEFAULT_BAUD}`}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 py-1 text-xs font-semibold rounded-full ${device.is_active ? 'bg-[#DCFCE7] text-[#065F46]' : 'bg-gray-100 text-gray-600'}`}>
                      {device.is_active ? 'Activo' : 'Inactivo'}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm flex gap-3">
                    <button
                      onClick={() => handleTrigger(device)}
                      disabled={!device.is_active || !canTrigger(device) || triggering[device.id]}
                      className="text-[#1272D6] hover:text-[#0A4A8F] font-medium disabled:opacity-40 disabled:cursor-not-allowed"
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
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Ubicación</label>
            <input
              type="text"
              value={form.location}
              onChange={e => setForm({ ...form, location: e.target.value })}
              placeholder="Ej: Entrada principal"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Tipo de dispositivo</label>
            <select
              value={form.device_type}
              onChange={e => setForm({ ...form, device_type: e.target.value })}
              disabled={!!editing}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent disabled:bg-gray-100 disabled:text-gray-500"
            >
              <option value={DEVICE_TYPE_RELAY}>Relé por puerto serial (Arduino)</option>
              <option value={DEVICE_TYPE_HIKVISION}>Terminal Hikvision (ISAPI, red)</option>
            </select>
            {editing && (
              <p className="text-xs text-gray-400 mt-1">
                El tipo no se puede cambiar una vez creado — elimina y crea uno nuevo si es necesario.
              </p>
            )}
          </div>

          {form.device_type === DEVICE_TYPE_HIKVISION ? (
            <>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Dirección IP *</label>
                  <input
                    type="text"
                    value={form.ip_address}
                    onChange={e => setForm({ ...form, ip_address: e.target.value })}
                    required
                    placeholder="Ej: 192.168.1.149"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent font-mono"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Puerto HTTP</label>
                  <input
                    type="number"
                    value={form.http_port}
                    onChange={e => setForm({ ...form, http_port: e.target.value })}
                    placeholder="80"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Usuario admin</label>
                  <input
                    type="text"
                    value={form.username}
                    onChange={e => setForm({ ...form, username: e.target.value })}
                    placeholder="admin"
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Contraseña {editing && <span className="text-gray-400 font-normal">(vacío = no cambiar)</span>}
                  </label>
                  <input
                    type="password"
                    value={form.password}
                    onChange={e => setForm({ ...form, password: e.target.value })}
                    placeholder={editing ? '••••••••' : ''}
                    required={!editing}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Número de puerta</label>
                <input
                  type="number"
                  min="1"
                  value={form.door_no}
                  onChange={e => setForm({ ...form, door_no: e.target.value })}
                  placeholder="1"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                />
                <p className="text-xs text-gray-400 mt-1">Los terminales de una sola puerta (como el DS-K1T8003MF) usan 1.</p>
              </div>
            </>
          ) : (
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Puerto COM</label>
                <input
                  type="text"
                  value={form.com_port}
                  onChange={e => setForm({ ...form, com_port: e.target.value })}
                  placeholder="Ej: COM5"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent font-mono"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Baud Rate</label>
                <select
                  value={form.baud_rate}
                  onChange={e => setForm({ ...form, baud_rate: Number(e.target.value) })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                >
                  {[9600, 19200, 38400, 57600, 115200].map(b => (
                    <option key={b} value={b}>{b}</option>
                  ))}
                </select>
              </div>
            </div>
          )}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Notas</label>
            <input
              type="text"
              value={form.notes}
              onChange={e => setForm({ ...form, notes: e.target.value })}
              placeholder="Descripción opcional"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
            />
          </div>
          <div className="flex gap-3 pt-2">
            <button type="button" onClick={() => setShowModal(false)} className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-xl hover:bg-gray-50 transition">
              Cancelar
            </button>
            <button type="submit" disabled={saving} className="flex-1 px-4 py-2 bg-[#1272D6] text-white rounded-xl  transition disabled:opacity-50">
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
