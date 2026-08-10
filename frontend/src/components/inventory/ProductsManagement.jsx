import { useState, useEffect } from 'react';
import api from '../../utils/api';
import Toast from '../Toast';
import ConfirmDialog from '../ConfirmDialog';
import Modal from '../Modal';
import { fmt } from '../../utils/currency';

export default function ProductsManagement() {
  const [products, setProducts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [showModal, setShowModal] = useState(false);
  const [editingProduct, setEditingProduct] = useState(null);
  const [toast, setToast] = useState(null);
  const [confirmDialog, setConfirmDialog] = useState({ open: false, onConfirm: null, message: '' });
  const [showStockModal, setShowStockModal] = useState(false);
  const [stockProduct, setStockProduct] = useState(null);
  const [newStock, setNewStock] = useState('');
  // Dos formas distintas de mover inventario, y conviene no confundirlas:
  // 'count' es un conteo fisico (cuantas hay de verdad) y 'move' es una entrada
  // o salida concreta (llego mercancia, se rompio una). El backend recibe un
  // valor absoluto en el primer caso y un delta con motivo en el segundo.
  const [stockMode, setStockMode] = useState('count');
  const [moveDelta, setMoveDelta] = useState('');
  const [moveDirection, setMoveDirection] = useState('in');
  const [moveReason, setMoveReason] = useState('adjustment');
  const [moveNote, setMoveNote] = useState('');
  const [savingStock, setSavingStock] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    unit_price: '',
    stock: '',
    min_stock: '',
    status: 'active'
  });

  useEffect(() => {
    fetchProducts();
  }, []);

  const fetchProducts = async () => {
    setLoading(true);
    try {
      const response = await api.get('/products');
      if (response.ok) {
        const data = await response.json();
        setProducts(data || []);
      }
    } catch (error) {
      console.error('Error fetching products:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async (query) => {
    setSearchQuery(query);
    if (!query.trim()) {
      fetchProducts();
      return;
    }

    try {
      const response = await api.get(`/products/search?q=${encodeURIComponent(query)}`);
      if (response.ok) {
        const data = await response.json();
        setProducts(data || []);
      }
    } catch (error) {
      console.error('Error searching products:', error);
    }
  };

  const openCreateModal = () => {
    setEditingProduct(null);
    setFormData({
      name: '',
      description: '',
      unit_price: '',
      stock: '',
      min_stock: '',
      status: 'active'
    });
    setShowModal(true);
  };

  const openEditModal = (product) => {
    setEditingProduct(product);
    setFormData({
      name: product.name,
      description: product.description || '',
      unit_price: product.unit_price,
      stock: product.stock,
      min_stock: product.min_stock ?? '',
      status: product.status
    });
    setShowModal(true);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    // `stock` solo va en el alta. Editar un producto no mueve inventario: el
    // stock pertenece al kardex y solo cambia por conteo o por entrada/salida,
    // que dejan asiento. El backend tampoco lo aceptaria aqui.
    const payload = {
      name: formData.name,
      description: formData.description,
      unit_price: parseFloat(formData.unit_price),
      min_stock: parseInt(formData.min_stock) || 0,
      status: formData.status
    };
    if (!editingProduct) {
      payload.stock = parseInt(formData.stock) || 0;
    }

    try {
      const response = editingProduct
        ? await api.put(`/products/${editingProduct.id}`, payload)
        : await api.post('/products', payload);

      if (response.ok) {
        setShowModal(false);
        setToast({ message: editingProduct ? 'Producto actualizado' : 'Producto creado exitosamente', type: 'success' });
        fetchProducts();
      } else {
        const error = await response.json();
        setToast({ message: error.error || 'Error al guardar el producto', type: 'error' });
      }
    } catch (error) {
      console.error('Error saving product:', error);
      setToast({ message: 'Error al guardar el producto', type: 'error' });
    }
  };

  const handleDelete = (id) => {
    setConfirmDialog({
      open: true,
      message: '¿Estás seguro de eliminar este producto? Esta acción no se puede deshacer.',
      onConfirm: async () => {
        try {
          const response = await api.delete(`/products/${id}`);
          if (response.ok) {
            setToast({ message: 'Producto eliminado', type: 'success' });
            fetchProducts();
          }
        } catch (error) {
          console.error('Error deleting product:', error);
          setToast({ message: 'Error al eliminar producto', type: 'error' });
        }
      },
    });
  };

  const openStockModal = (product) => {
    setStockProduct(product);
    setNewStock(String(product.stock));
    setStockMode('count');
    setMoveDelta('');
    setMoveDirection('in');
    setMoveReason('adjustment');
    setMoveNote('');
    setShowStockModal(true);
  };

  const handleUpdateStock = async () => {
    if (!stockProduct || savingStock) return;
    setSavingStock(true);
    try {
      let response;
      if (stockMode === 'count') {
        response = await api.patch(`/products/${stockProduct.id}/stock`, {
          stock: parseInt(newStock),
          note: moveNote,
        });
      } else {
        const magnitude = Math.abs(parseInt(moveDelta) || 0);
        if (magnitude === 0) {
          setToast({ message: 'Indica cuantas unidades entran o salen', type: 'error' });
          return;
        }
        response = await api.post(`/products/${stockProduct.id}/stock/adjust`, {
          delta: moveDirection === 'out' ? -magnitude : magnitude,
          reason: moveReason,
          note: moveNote,
        });
      }

      if (response.ok) {
        setToast({ message: 'Inventario actualizado', type: 'success' });
        setShowStockModal(false);
        fetchProducts();
      } else {
        // El caso real: intentar sacar mas unidades de las que hay. El backend lo
        // rechaza en el WHERE del UPDATE, y el mensaje que manda es el util.
        const body = await response.json().catch(() => ({}));
        setToast({ message: body.message || 'No se pudo actualizar el inventario', type: 'error' });
      }
    } catch (error) {
      console.error('Error updating stock:', error);
      setToast({ message: 'Error al actualizar inventario', type: 'error' });
    } finally {
      setSavingStock(false);
    }
  };

  // El umbral lo decide cada producto (min_stock), no un 10 cableado que trataba
  // igual una barra de proteina y un suplemento de doscientos mil pesos.
  const getStockBadge = (product) => {
    if (product.stock === 0) {
      return <span className="px-2 py-1 text-xs font-semibold bg-red-100 text-red-800 rounded-full">Sin Stock</span>;
    } else if (product.low_stock) {
      return <span className="px-2 py-1 text-xs font-semibold bg-yellow-100 text-yellow-800 rounded-full">Stock Bajo</span>;
    } else {
      return <span className="px-2 py-1 text-xs font-semibold bg-[#DCFCE7] text-[#065F46] rounded-full">Disponible</span>;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">Gestión de Productos</h2>
          <p className="text-gray-600 mt-1">Administra el inventario de productos</p>
        </div>
        <button
          onClick={openCreateModal}
          className="px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition flex items-center space-x-2"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          <span>Nuevo Producto</span>
        </button>
      </div>

      {/* Search Bar */}
      <div className="bg-white rounded-lg shadow-md p-4">
        <div className="relative">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => handleSearch(e.target.value)}
            placeholder="Buscar productos por nombre..."
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
          />
          <svg
            className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
      </div>

      {/* Products Table */}
      <div className="bg-white rounded-lg shadow-md overflow-hidden">
        {loading ? (
          <div className="p-8 text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-[#1272D6] mx-auto"></div>
            <p className="text-gray-600 mt-4">Cargando productos...</p>
          </div>
        ) : products.length === 0 ? (
          <div className="p-8 text-center">
            <p className="text-gray-600">No se encontraron productos</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Producto
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Precio
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Stock
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Estado
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Acciones
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {products.map((product) => (
                  <tr key={product.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <div className="flex flex-col">
                        <span className="text-sm font-medium text-gray-900">{product.name}</span>
                        {product.description && (
                          <span className="text-xs text-gray-500">{product.description}</span>
                        )}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="text-sm font-semibold text-gray-900">
                        {fmt(product.unit_price)}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center space-x-2">
                        <span className="text-sm text-gray-900">{product.stock}</span>
                        {getStockBadge(product)}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`px-2 py-1 text-xs font-semibold rounded-full ${
                          product.status === 'active'
                            ? 'bg-[#DCFCE7] text-[#065F46]'
                            : 'bg-gray-100 text-gray-800'
                        }`}
                      >
                        {product.status === 'active' ? 'Activo' : 'Inactivo'}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <div className="flex items-center space-x-2">
                        <button
                          onClick={() => openStockModal(product)}
                          className="text-[#1272D6] hover:text-[#0A4A8F] font-medium"
                          title="Actualizar stock"
                        >
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
                          </svg>
                        </button>
                        <button
                          onClick={() => openEditModal(product)}
                          className="text-[#1272D6] hover:text-[#0A4A8F] font-medium"
                          title="Editar"
                        >
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                          </svg>
                        </button>
                        <button
                          onClick={() => handleDelete(product.id)}
                          className="text-red-600 hover:text-red-900 font-medium"
                          title="Eliminar"
                        >
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/25 backdrop-blur-sm flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full">
            <div className="p-6">
              <h3 className="text-xl font-bold text-gray-900 mb-4">
                {editingProduct ? 'Editar Producto' : 'Nuevo Producto'}
              </h3>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Nombre *
                  </label>
                  <input
                    type="text"
                    required
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Descripción
                  </label>
                  <textarea
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                    rows={3}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Precio Unitario *
                  </label>
                  <input
                    type="number"
                    required
                    step="0.01"
                    min="0"
                    value={formData.unit_price}
                    onChange={(e) => setFormData({ ...formData, unit_price: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  />
                </div>

                {/*
                  El stock inicial solo se pide al crear. Editando, el campo
                  desaparece a proposito: el stock lo mueve el kardex (conteo o
                  entrada/salida), que deja asiento de quien lo cambio y por que.
                */}
                {!editingProduct && (
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Stock Inicial *
                    </label>
                    <input
                      type="number"
                      required
                      min="0"
                      value={formData.stock}
                      onChange={(e) => setFormData({ ...formData, stock: e.target.value })}
                      className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                    />
                  </div>
                )}

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Alerta de stock bajo
                  </label>
                  <input
                    type="number"
                    min="0"
                    placeholder="0 = sin alerta"
                    value={formData.min_stock}
                    onChange={(e) => setFormData({ ...formData, min_stock: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  />
                  <p className="mt-1 text-xs text-gray-400">
                    Se marca &laquo;Stock Bajo&raquo; cuando queden estas unidades o menos.
                  </p>
                </div>

                {editingProduct && (
                  <p className="text-xs text-gray-400 bg-gray-50 border border-gray-200 rounded-lg px-3 py-2">
                    El stock actual ({editingProduct.stock}) se cambia desde el boton de inventario
                    de la fila, para que quede registrado en el historial de movimientos.
                  </p>
                )}

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Estado
                  </label>
                  <select
                    value={formData.status}
                    onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  >
                    <option value="active">Activo</option>
                    <option value="inactive">Inactivo</option>
                  </select>
                </div>

                <div className="flex space-x-3 pt-4">
                  <button
                    type="button"
                    onClick={() => setShowModal(false)}
                    className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition"
                  >
                    Cancelar
                  </button>
                  <button
                    type="submit"
                    className="flex-1 px-4 py-2 bg-[#1272D6] text-white rounded-lg hover:bg-[#0D5BAD] transition"
                  >
                    {editingProduct ? 'Actualizar' : 'Crear'}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* Stock Update Modal */}
      <Modal
        isOpen={showStockModal && !!stockProduct}
        onClose={() => setShowStockModal(false)}
        title="Movimiento de Inventario"
        maxWidth="sm"
      >
        {stockProduct && (
          <div className="space-y-4">
            <p className="text-sm text-gray-600">
              Producto: <span className="font-semibold text-gray-900">{stockProduct.name}</span>
              <span className="text-gray-400"> &middot; hay {stockProduct.stock}</span>
            </p>

            {/* Conteo y movimiento son operaciones distintas y se piden distinto */}
            <div className="grid grid-cols-2 gap-2">
              {[
                { key: 'move', label: 'Entrada / Salida' },
                { key: 'count', label: 'Conteo fisico' },
              ].map(({ key, label }) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => setStockMode(key)}
                  className={`px-3 py-2 text-sm font-semibold rounded-lg border transition ${
                    stockMode === key
                      ? 'border-[#1272D6] bg-[#EBF3FF] text-[#1272D6]'
                      : 'border-gray-200 text-gray-600 hover:bg-gray-50'
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>

            {stockMode === 'move' ? (
              <>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={() => { setMoveDirection('in'); setMoveReason('adjustment'); }}
                    className={`px-3 py-2 text-sm font-semibold rounded-lg border transition ${
                      moveDirection === 'in'
                        ? 'border-[#059669] bg-[#DCFCE7] text-[#065F46]'
                        : 'border-gray-200 text-gray-600 hover:bg-gray-50'
                    }`}
                  >
                    &#43; Entrada
                  </button>
                  <button
                    type="button"
                    onClick={() => { setMoveDirection('out'); setMoveReason('shrinkage'); }}
                    className={`px-3 py-2 text-sm font-semibold rounded-lg border transition ${
                      moveDirection === 'out'
                        ? 'border-red-500 bg-red-50 text-red-700'
                        : 'border-gray-200 text-gray-600 hover:bg-gray-50'
                    }`}
                  >
                    &minus; Salida
                  </button>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Unidades</label>
                  <input
                    type="number"
                    min="1"
                    value={moveDelta}
                    onChange={(e) => setMoveDelta(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                    autoFocus
                  />
                  {moveDelta && (
                    <p className="mt-1 text-xs text-gray-500">
                      Quedaran{' '}
                      <span className="font-semibold text-gray-800">
                        {moveDirection === 'out'
                          ? stockProduct.stock - Math.abs(parseInt(moveDelta) || 0)
                          : stockProduct.stock + Math.abs(parseInt(moveDelta) || 0)}
                      </span>{' '}
                      unidades.
                    </p>
                  )}
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Motivo</label>
                  <select
                    value={moveReason}
                    onChange={(e) => setMoveReason(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  >
                    <option value="adjustment">Ajuste manual</option>
                    <option value="shrinkage">Merma (vencido, roto, robo)</option>
                  </select>
                </div>
              </>
            ) : (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Unidades contadas
                </label>
                <input
                  type="number"
                  min="0"
                  value={newStock}
                  onChange={(e) => setNewStock(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
                  autoFocus
                />
                {newStock !== '' && parseInt(newStock) !== stockProduct.stock && (
                  <p className="mt-1 text-xs text-gray-500">
                    Se registrara una diferencia de{' '}
                    <span className="font-semibold text-gray-800">
                      {parseInt(newStock) - stockProduct.stock > 0 ? '+' : ''}
                      {parseInt(newStock) - stockProduct.stock}
                    </span>{' '}
                    unidades.
                  </p>
                )}
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Nota <span className="text-gray-400 font-normal">(opcional)</span>
              </label>
              <input
                type="text"
                value={moveNote}
                onChange={(e) => setMoveNote(e.target.value)}
                placeholder="Por que se mueve"
                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-[#1272D6] focus:border-transparent"
              />
            </div>

            <p className="text-xs text-gray-400">
              El movimiento queda registrado con tu nombre en el historial de inventario.
            </p>

            <div className="flex gap-3">
              <button
                onClick={() => setShowStockModal(false)}
                className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-xl hover:bg-gray-50 transition"
              >
                Cancelar
              </button>
              <button
                onClick={handleUpdateStock}
                disabled={savingStock}
                className="flex-1 px-4 py-2 bg-[#1272D6] text-white rounded-xl transition disabled:opacity-50"
              >
                {savingStock ? 'Guardando...' : 'Registrar'}
              </button>
            </div>
          </div>
        )}
      </Modal>

      <ConfirmDialog
        isOpen={confirmDialog.open}
        onClose={() => setConfirmDialog({ open: false, onConfirm: null, message: '' })}
        onConfirm={confirmDialog.onConfirm}
        title="Eliminar producto"
        message={confirmDialog.message}
        confirmText="Eliminar"
        variant="danger"
      />

      {toast && (
        <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />
      )}
    </div>
  );
}
