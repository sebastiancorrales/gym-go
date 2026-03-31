import { useState, useEffect } from 'react';
import api from '../../utils/api';
import Toast from '../Toast';
import Modal from '../Modal';
import EmptyState from '../EmptyState';
import { fmt } from '../../utils/currency';

export default function SalesTab({ user }) {
  const [products, setProducts] = useState([]);
  const [paymentMethods, setPaymentMethods] = useState([]);
  const [cart, setCart] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [splitPayments, setSplitPayments] = useState([{ payment_method_id: '', amount: '' }]);
  const [receivedCash, setReceivedCash] = useState('');
  const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState(null);

  useEffect(() => {
    fetchProducts();
    fetchPaymentMethods();
  }, []);

  const fetchProducts = async () => {
    try {
      const response = await api.get('/products?status=active');
      if (response.ok) {
        const data = await response.json();
        setProducts(data || []);
      }
    } catch (error) {
      console.error('Error fetching products:', error);
    }
  };

  const fetchPaymentMethods = async () => {
    try {
      const response = await api.get('/payment-methods?status=active');
      if (response.ok) {
        const data = await response.json();
        setPaymentMethods(data || []);
        
        // Auto-select "Efectivo" if available
        const efectivo = data.find(method =>
          method.name.toLowerCase().includes('efectivo') ||
          method.name.toLowerCase().includes('cash')
        );
        const defaultId = efectivo ? efectivo.id : (data.length > 0 ? data[0].id : '');
        setSplitPayments([{ payment_method_id: defaultId, amount: '' }]);
      }
    } catch (error) {
      console.error('Error fetching payment methods:', error);
    }
  };

  const handleSearch = async (query) => {
    setSearchQuery(query);
    if (!query.trim()) {
      fetchProducts();
      return;
    }

    try {
      const response = await api.get(`/products/search?q=${encodeURIComponent(query)}&status=active`);
      if (response.ok) {
        const data = await response.json();
        setProducts(data || []);
      }
    } catch (error) {
      console.error('Error searching products:', error);
    }
  };

  const addToCart = (product) => {
    const existingItem = cart.find(item => item.product_id === product.id);
    
    if (existingItem) {
      // Check stock limit
      if (existingItem.quantity >= product.stock) {
        setToast({ message: 'No hay suficiente stock disponible', type: 'warning' });
        return;
      }
      setCart(cart.map(item =>
        item.product_id === product.id
          ? { ...item, quantity: item.quantity + 1 }
          : item
      ));
    } else {
      if (product.stock === 0) {
        setToast({ message: 'Producto sin stock', type: 'warning' });
        return;
      }
      setCart([...cart, {
        product_id: product.id,
        product_name: product.name,
        unit_price: product.unit_price,
        quantity: 1,
        discount: 0,
        max_stock: product.stock
      }]);
    }
  };

  const updateCartItem = (productId, field, value) => {
    setCart(cart.map(item => {
      if (item.product_id === productId) {
        const updatedItem = { ...item, [field]: value };
        
        // Validate quantity against stock
        if (field === 'quantity' && value > item.max_stock) {
          setToast({ message: `Solo hay ${item.max_stock} unidades disponibles`, type: 'warning' });
          return item;
        }
        
        return updatedItem;
      }
      return item;
    }));
  };

  const removeFromCart = (productId) => {
    setCart(cart.filter(item => item.product_id !== productId));
  };

  const calculateItemSubtotal = (item) => {
    const total = item.quantity * item.unit_price;
    return total - item.discount;
  };

  const calculateTotals = () => {
    const subtotal = cart.reduce((sum, item) => sum + (item.quantity * item.unit_price), 0);
    const totalDiscount = cart.reduce((sum, item) => sum + item.discount, 0);
    const total = subtotal - totalDiscount;
    return { subtotal, totalDiscount, total };
  };

  const isCashMethod = (id) => {
    const m = paymentMethods.find(p => p.id === id);
    return m && (m.type === 'cash' || m.name.toLowerCase().includes('efectivo'));
  };

  const splitTotal = splitPayments.reduce((s, p) => s + (parseFloat(p.amount) || 0), 0);
  const splitPending = totals.total - splitTotal;
  const singleIsCash = splitPayments.length === 1 && isCashMethod(splitPayments[0].payment_method_id);
  const change = singleIsCash ? (parseFloat(receivedCash) || 0) - totals.total : 0;

  const addSplitRow = () =>
    setSplitPayments(prev => [...prev, { payment_method_id: paymentMethods[0]?.id || '', amount: '' }]);

  const removeSplitRow = (idx) =>
    setSplitPayments(prev => prev.filter((_, i) => i !== idx));

  const updateSplitRow = (idx, field, value) =>
    setSplitPayments(prev => prev.map((p, i) => i === idx ? { ...p, [field]: value } : p));

  const handleCheckout = () => {
    if (cart.length === 0) {
      setToast({ message: 'El carrito está vacío', type: 'warning' });
      return;
    }
    // Pre-fill single payment amount with total
    setSplitPayments(prev => {
      if (prev.length === 1 && prev[0].amount === '') {
        return [{ ...prev[0], amount: String(totals.total) }];
      }
      return prev;
    });
    setReceivedCash('');
    setShowPaymentModal(true);
  };

  const completeSale = async () => {
    const validPayments = splitPayments.filter(p => p.payment_method_id && parseFloat(p.amount) > 0);
    if (validPayments.length === 0) {
      setToast({ message: 'Seleccione un método de pago', type: 'warning' });
      return;
    }
    if (Math.abs(splitPending) > 0.01) {
      setToast({ message: `Falta asignar ${fmt(splitPending)} al método de pago`, type: 'warning' });
      return;
    }

    setLoading(true);
    try {
      const saleData = {
        payments: validPayments.map(p => ({
          payment_method_id: p.payment_method_id,
          amount: parseFloat(p.amount)
        })),
        details: cart.map(item => ({
          product_id: item.product_id,
          quantity: item.quantity,
          unit_price: item.unit_price,
          discount: item.discount
        }))
      };

      const response = await api.post('/sales', saleData);

      if (response.ok) {
        const data = await response.json();
        setToast({
          message: `Venta completada - Total: ${fmt(data.total)}`,
          type: 'success'
        });

        setCart([]);
        setShowPaymentModal(false);
        setReceivedCash('');
        fetchProducts();

        const defaultId = paymentMethods.find(m =>
          m.name.toLowerCase().includes('efectivo') || m.name.toLowerCase().includes('cash')
        )?.id || paymentMethods[0]?.id || '';
        setSplitPayments([{ payment_method_id: defaultId, amount: '' }]);
      } else {
        const error = await response.json();
        setToast({ message: error.error || 'Error al completar la venta', type: 'error' });
      }
    } catch (error) {
      console.error('Error completing sale:', error);
      setToast({ message: 'Error al completar la venta', type: 'error' });
    } finally {
      setLoading(false);
    }
  };

  const totals = calculateTotals();

  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      {/* Products Section */}
      <div className="lg:col-span-2 space-y-4">
        <div>
          <h2 className="text-2xl font-extrabold text-gray-900 mb-4">Punto de Venta</h2>
          
          {/* Search */}
          <div className="relative mb-4">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => handleSearch(e.target.value)}
              placeholder="Buscar productos..."
              className="w-full pl-10 pr-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
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

        {/* Products Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
          {products.filter(p => p.stock > 0).map((product) => (
            <div
              key={product.id}
              onClick={() => addToCart(product)}
              className="group relative bg-white rounded-2xl border border-gray-100 p-4 cursor-pointer hover:border-emerald-200 hover:shadow-lg hover:shadow-emerald-500/5 transition-all duration-200"
            >
              <div className="flex flex-col h-full">
                <div className="flex-1">
                  <div className="flex items-start justify-between mb-2">
                    <div className="w-10 h-10 bg-gradient-to-br from-emerald-50 to-cyan-50 rounded-xl flex items-center justify-center flex-shrink-0">
                      <svg className="w-5 h-5 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                      </svg>
                    </div>
                    <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${
                      product.stock <= 5 ? 'bg-amber-100 text-amber-700' : 'bg-emerald-100 text-emerald-700'
                    }`}>
                      {product.stock} uds
                    </span>
                  </div>
                  <h3 className="font-bold text-gray-900 mb-0.5">{product.name}</h3>
                  {product.description && (
                    <p className="text-xs text-gray-400 line-clamp-2">{product.description}</p>
                  )}
                </div>
                <div className="flex items-center justify-between mt-3 pt-3 border-t border-gray-50">
                  <span className="text-lg font-extrabold text-emerald-600">
                    {fmt(product.unit_price)}
                  </span>
                  <div className="w-8 h-8 bg-emerald-500 rounded-lg flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity shadow-md">
                    <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 4v16m8-8H4" />
                    </svg>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>

        {products.length === 0 && (
          <div className="bg-white rounded-2xl">
            <EmptyState icon="box" title="No hay productos disponibles" description="Agrega productos desde la seccion de inventario" />
          </div>
        )}
      </div>

      {/* Cart Section */}
      <div className="lg:col-span-1">
        <div className="bg-white rounded-2xl border border-gray-100 shadow-sm p-6 sticky top-6">
          <h3 className="text-lg font-extrabold text-gray-900 mb-4 flex items-center">
            <div className="w-8 h-8 bg-gradient-to-br from-emerald-500 to-cyan-500 rounded-lg flex items-center justify-center mr-2 shadow-md shadow-emerald-500/20">
              <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 11-4 0 2 2 0 014 0z" />
              </svg>
            </div>
            Carrito
          </h3>

          {cart.length === 0 ? (
            <div className="text-center py-10">
              <div className="w-14 h-14 bg-gray-100 rounded-2xl flex items-center justify-center mx-auto mb-3">
                <svg className="w-7 h-7 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 11-4 0 2 2 0 014 0z" />
                </svg>
              </div>
              <p className="text-sm font-medium text-gray-400">Carrito vacio</p>
              <p className="text-xs text-gray-300 mt-1">Selecciona productos para comenzar</p>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Cart Items */}
              <div className="space-y-3 max-h-96 overflow-y-auto">
                {cart.map((item) => (
                  <div key={item.product_id} className="border border-gray-200 rounded-lg p-3">
                    <div className="flex justify-between items-start mb-2">
                      <span className="font-medium text-sm text-gray-900">{item.product_name}</span>
                      <button
                        onClick={() => removeFromCart(item.product_id)}
                        className="text-red-500 hover:text-red-700"
                      >
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    </div>

                    <div className="grid grid-cols-2 gap-2 text-xs">
                      <div>
                        <label className="text-gray-600">Cantidad</label>
                        <input
                          type="number"
                          min="1"
                          max={item.max_stock}
                          value={item.quantity}
                          onChange={(e) => updateCartItem(item.product_id, 'quantity', parseInt(e.target.value) || 1)}
                          className="w-full px-2 py-1 border border-gray-300 rounded"
                        />
                      </div>
                      <div>
                        <label className="text-gray-600">Descuento $</label>
                        <input
                          type="number"
                          min="0"
                          step="0.01"
                          value={item.discount}
                          onChange={(e) => updateCartItem(item.product_id, 'discount', parseFloat(e.target.value) || 0)}
                          className="w-full px-2 py-1 border border-gray-300 rounded"
                        />
                      </div>
                    </div>

                    <div className="mt-2 text-right">
                      <span className="text-xs text-gray-600">Precio: {fmt(item.unit_price)}</span>
                      <br />
                      <span className="font-semibold text-blue-600">
                        Subtotal: {fmt(calculateItemSubtotal(item))}
                      </span>
                    </div>
                  </div>
                ))}
              </div>

              {/* Totals */}
              <div className="border-t border-gray-200 pt-4 space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-600">Subtotal:</span>
                  <span className="font-medium">{fmt(totals.subtotal)}</span>
                </div>
                {totals.totalDiscount > 0 && (
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-600">Descuentos:</span>
                    <span className="font-medium text-red-600">-{fmt(totals.totalDiscount)}</span>
                  </div>
                )}
                <div className="flex justify-between text-lg font-bold pt-2 border-t">
                  <span>Total:</span>
                  <span className="text-blue-600">{fmt(totals.total)}</span>
                </div>
              </div>

              {/* Checkout Button */}
              <button
                onClick={handleCheckout}
                className="w-full py-3 bg-gradient-to-r from-emerald-500 to-cyan-500 text-white rounded-xl hover:from-emerald-600 hover:to-cyan-600 transition font-bold shadow-lg shadow-emerald-500/20"
              >
                Procesar Venta
              </button>

              <button
                onClick={() => setCart([])}
                className="w-full py-2 text-gray-400 hover:text-gray-600 rounded-xl transition text-sm font-medium"
              >
                Limpiar Carrito
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Payment Modal */}
      <Modal
        isOpen={showPaymentModal}
        onClose={() => setShowPaymentModal(false)}
        title="Finalizar Venta"
        maxWidth="md"
      >
        <div className="space-y-4 mb-6">
          {/* Totals summary */}
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="flex justify-between text-sm mb-2">
              <span className="text-gray-600">Subtotal:</span>
              <span className="font-medium">{fmt(totals.subtotal)}</span>
            </div>
            {totals.totalDiscount > 0 && (
              <div className="flex justify-between text-sm mb-2">
                <span className="text-gray-600">Descuentos:</span>
                <span className="font-medium text-red-600">-{fmt(totals.totalDiscount)}</span>
              </div>
            )}
            <div className="flex justify-between text-lg font-bold pt-2 border-t border-gray-300">
              <span>Total a Pagar:</span>
              <span className="text-blue-600">{fmt(totals.total)}</span>
            </div>
          </div>

          {/* Split payment rows */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-sm font-medium text-gray-700">Forma de pago</label>
              <button
                type="button"
                onClick={addSplitRow}
                className="text-xs text-emerald-600 hover:text-emerald-800 font-medium"
              >
                + Agregar método
              </button>
            </div>
            <div className="space-y-2">
              {splitPayments.map((row, idx) => (
                <div key={idx} className="flex gap-2 items-center">
                  <select
                    value={row.payment_method_id}
                    onChange={e => updateSplitRow(idx, 'payment_method_id', e.target.value)}
                    className="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                  >
                    {paymentMethods.map(m => (
                      <option key={m.id} value={m.id}>{m.name}</option>
                    ))}
                  </select>
                  <input
                    type="number"
                    min="0"
                    step="100"
                    placeholder="Monto"
                    value={row.amount}
                    onChange={e => updateSplitRow(idx, 'amount', e.target.value)}
                    className="w-32 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
                  />
                  {splitPayments.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeSplitRow(idx)}
                      className="text-red-400 hover:text-red-600 px-1"
                    >
                      ✕
                    </button>
                  )}
                </div>
              ))}
            </div>

            {/* Pending balance indicator */}
            {splitPayments.length > 1 && (
              <div className={`mt-2 text-sm font-medium flex justify-between ${Math.abs(splitPending) < 0.01 ? 'text-emerald-600' : 'text-amber-600'}`}>
                <span>{Math.abs(splitPending) < 0.01 ? 'Cubierto' : 'Pendiente:'}</span>
                {Math.abs(splitPending) >= 0.01 && <span>{fmt(splitPending)}</span>}
              </div>
            )}
          </div>

          {/* Vuelto calculator (solo si un único método efectivo) */}
          {singleIsCash && (
            <div className="bg-amber-50 border border-amber-200 rounded-lg p-3">
              <label className="block text-sm font-medium text-amber-800 mb-1">
                Monto recibido
              </label>
              <input
                type="number"
                min="0"
                step="1000"
                value={receivedCash}
                onChange={e => setReceivedCash(e.target.value)}
                placeholder={fmt(totals.total)}
                className="w-full px-3 py-2 border border-amber-300 rounded-lg text-sm focus:ring-2 focus:ring-amber-400 focus:border-transparent bg-white"
              />
              {receivedCash && change >= 0 && (
                <div className="mt-2 flex justify-between text-sm font-bold text-amber-900">
                  <span>Vuelto:</span>
                  <span className="text-lg">{fmt(change)}</span>
                </div>
              )}
              {receivedCash && change < 0 && (
                <p className="mt-1 text-xs text-red-600">El monto recibido es menor al total</p>
              )}
            </div>
          )}
        </div>

        <div className="flex space-x-3">
          <button
            type="button"
            onClick={() => setShowPaymentModal(false)}
            disabled={loading}
            className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition"
          >
            Cancelar
          </button>
          <button
            onClick={completeSale}
            disabled={loading || Math.abs(splitPending) > 0.01}
            className="flex-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? 'Procesando...' : 'Confirmar Venta'}
          </button>
        </div>
      </Modal>

      {/* Toast Notification */}
      {toast && (
        <Toast
          message={toast.message}
          type={toast.type}
          onClose={() => setToast(null)}
        />
      )}
    </div>
  );
}
