import { useState, useEffect } from 'react';
import api from '../../utils/api';

// Helper function to get today's date in YYYY-MM-DD format
const getTodayDate = () => {
  const today = new Date();
  const year = today.getFullYear();
  const month = String(today.getMonth() + 1).padStart(2, '0');
  const day = String(today.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

export default function SalesHistory() {
  const todayDate = getTodayDate();
  const [sales, setSales] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedSale, setSelectedSale] = useState(null);
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [dateRange, setDateRange] = useState({
    start_date: todayDate,
    end_date: todayDate
  });

  useEffect(() => {
    fetchSales();
  }, []);

  const fetchSales = async () => {
    setLoading(true);
    try {
      let endpoint = '/sales';
      
      // Add date filters if provided
      if (dateRange.start_date && dateRange.end_date) {
        endpoint = `/sales/by-date?start_date=${dateRange.start_date}&end_date=${dateRange.end_date}`;
      }

      const response = await api.get(endpoint);
      if (response.ok) {
        const data = await response.json();
        setSales(data || []);
      }
    } catch (error) {
      console.error('Error fetching sales:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleFilterByDate = (e) => {
    e.preventDefault();
    fetchSales();
  };

  const viewSaleDetail = async (saleId) => {
    try {
      const response = await api.get(`/sales/${saleId}`);
      if (response.ok) {
        const data = await response.json();
        setSelectedSale(data);
        setShowDetailModal(true);
      }
    } catch (error) {
      console.error('Error fetching sale detail:', error);
    }
  };

  const handleVoidSale = async (saleId) => {
    if (!confirm('¿Está seguro de anular esta venta? Esta acción no se puede deshacer.')) {
      return;
    }

    try {
      const response = await api.post(`/sales/${saleId}/void`, {});
      
      if (response.ok) {
        alert('Venta anulada exitosamente');
        setShowDetailModal(false);
        fetchSales();
      } else {
        const error = await response.json();
        alert(error.error || 'Error al anular la venta');
      }
    } catch (error) {
      console.error('Error voiding sale:', error);
      alert('Error al anular la venta');
    }
  };

  const getStatusBadge = (status) => {
    const badges = {
      completed: <span className="px-2 py-1 text-xs font-semibold bg-green-100 text-green-800 rounded-full">Completada</span>,
      voided: <span className="px-2 py-1 text-xs font-semibold bg-red-100 text-red-800 rounded-full">Anulada</span>,
      pending: <span className="px-2 py-1 text-xs font-semibold bg-yellow-100 text-yellow-800 rounded-full">Pendiente</span>
    };
    return badges[status] || <span className="px-2 py-1 text-xs font-semibold bg-gray-100 text-gray-800 rounded-full">{status}</span>;
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleString('es-ES', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-800">Historial de Ventas</h2>
        <p className="text-gray-600 mt-1">Consulta y gestiona las ventas realizadas</p>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg shadow-md p-4">
        <form onSubmit={handleFilterByDate} className="flex flex-wrap gap-4 items-end">
          <div className="flex-1 min-w-[200px]">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Fecha Inicio
            </label>
            <input
              type="date"
              value={dateRange.start_date}
              onChange={(e) => setDateRange({ ...dateRange, start_date: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <div className="flex-1 min-w-[200px]">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Fecha Fin
            </label>
            <input
              type="date"
              value={dateRange.end_date}
              onChange={(e) => setDateRange({ ...dateRange, end_date: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <div className="flex gap-2">
            <button
              type="submit"
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
            >
              Filtrar
            </button>
            <button
              type="button"
              onClick={() => {
                setDateRange({ start_date: '', end_date: '' });
                setTimeout(fetchSales, 0);
              }}
              className="px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition"
            >
              Limpiar
            </button>
          </div>
        </form>
      </div>

      {/* Sales Table */}
      <div className="bg-white rounded-lg shadow-md overflow-hidden">
        {loading ? (
          <div className="p-8 text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
            <p className="text-gray-600 mt-4">Cargando ventas...</p>
          </div>
        ) : sales.length === 0 ? (
          <div className="p-8 text-center">
            <p className="text-gray-600">No se encontraron ventas</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Fecha
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Total
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Descuentos
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Método de Pago
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
                {sales.map((sale) => (
                  <tr key={sale.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {formatDate(sale.sale_date)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`text-sm font-semibold ${sale.type === 'void' ? 'text-red-600' : 'text-green-600'}`}>
                        {sale.type === 'void' ? '-' : ''}${Math.abs(sale.total).toFixed(2)}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      ${sale.total_discount.toFixed(2)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {sale.payment_method_name || 'N/A'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {getStatusBadge(sale.status)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <button
                        onClick={() => viewSaleDetail(sale.id)}
                        className="text-blue-600 hover:text-blue-900 font-medium"
                      >
                        Ver Detalle
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Sale Detail Modal */}
      {showDetailModal && selectedSale && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6">
              <div className="flex items-center justify-between mb-6">
                <h3 className="text-xl font-bold text-gray-900">Detalle de Venta</h3>
                <button
                  onClick={() => setShowDetailModal(false)}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              {/* Sale Info */}
              <div className="bg-gray-50 rounded-lg p-4 mb-6 space-y-2">
                <div className="flex justify-between">
                  <span className="text-sm text-gray-600">Fecha:</span>
                  <span className="text-sm font-medium">{formatDate(selectedSale.sale_date)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm text-gray-600">Estado:</span>
                  {getStatusBadge(selectedSale.status)}
                </div>
                <div className="flex justify-between">
                  <span className="text-sm text-gray-600">Método de Pago:</span>
                  <span className="text-sm font-medium">{selectedSale.payment_method_name || 'N/A'}</span>
                </div>
                {selectedSale.type === 'void' && (
                  <div className="flex justify-between">
                    <span className="text-sm text-gray-600">Tipo:</span>
                    <span className="text-sm font-semibold text-red-600">Venta Anulada</span>
                  </div>
                )}
              </div>

              {/* Sale Details */}
              <div className="mb-6">
                <h4 className="font-semibold text-gray-900 mb-3">Productos</h4>
                <div className="space-y-2">
                  {selectedSale.details && selectedSale.details.map((detail, index) => (
                    <div key={index} className="flex justify-between items-start border-b border-gray-200 pb-2">
                      <div className="flex-1">
                        <p className="font-medium text-gray-900">{detail.product_name}</p>
                        <p className="text-sm text-gray-600">
                          {detail.quantity} x ${detail.unit_price.toFixed(2)}
                        </p>
                        {detail.discount > 0 && (
                          <p className="text-xs text-red-600">Descuento: -${detail.discount.toFixed(2)}</p>
                        )}
                      </div>
                      <span className="font-semibold text-gray-900">
                        ${detail.subtotal.toFixed(2)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>

              {/* Totals */}
              <div className="border-t border-gray-200 pt-4 space-y-2 mb-6">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-600">Subtotal:</span>
                  <span className="font-medium">
                    ${(selectedSale.total + selectedSale.total_discount).toFixed(2)}
                  </span>
                </div>
                {selectedSale.total_discount > 0 && (
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-600">Descuentos:</span>
                    <span className="font-medium text-red-600">
                      -${selectedSale.total_discount.toFixed(2)}
                    </span>
                  </div>
                )}
                <div className="flex justify-between text-lg font-bold pt-2 border-t">
                  <span>Total:</span>
                  <span className={selectedSale.type === 'void' ? 'text-red-600' : 'text-green-600'}>
                    {selectedSale.type === 'void' ? '-' : ''}${Math.abs(selectedSale.total).toFixed(2)}
                  </span>
                </div>
              </div>

              {/* Actions */}
              <div className="flex space-x-3">
                <button
                  onClick={() => setShowDetailModal(false)}
                  className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition"
                >
                  Cerrar
                </button>
                {selectedSale.status === 'completed' && selectedSale.type !== 'void' && (
                  <button
                    onClick={() => handleVoidSale(selectedSale.id)}
                    className="flex-1 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition"
                  >
                    Anular Venta
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
