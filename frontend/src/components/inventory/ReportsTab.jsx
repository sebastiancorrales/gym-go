import { useState } from 'react';
import api from '../../utils/api';

export default function ReportsTab() {
  const [activeReport, setActiveReport] = useState('sales');
  const [dateRange, setDateRange] = useState({
    start_date: '',
    end_date: ''
  });
  const [salesReport, setSalesReport] = useState(null);
  const [productReport, setProductReport] = useState(null);
  const [loading, setLoading] = useState(false);

  const fetchSalesReport = async (e) => {
    e.preventDefault();
    
    if (!dateRange.start_date || !dateRange.end_date) {
      alert('Por favor seleccione ambas fechas');
      return;
    }

    setLoading(true);
    try {
      const response = await api.get(
        `/sales/report?start_date=${dateRange.start_date}&end_date=${dateRange.end_date}`
      );
      
      if (response.ok) {
        const data = await response.json();
        setSalesReport(data);
      } else {
        alert('Error al generar el reporte');
      }
    } catch (error) {
      console.error('Error fetching sales report:', error);
      alert('Error al generar el reporte');
    } finally {
      setLoading(false);
    }
  };

  const fetchProductReport = async (e) => {
    e.preventDefault();
    
    if (!dateRange.start_date || !dateRange.end_date) {
      alert('Por favor seleccione ambas fechas');
      return;
    }

    setLoading(true);
    try {
      const response = await api.get(
        `/sales/report/by-product?start_date=${dateRange.start_date}&end_date=${dateRange.end_date}`
      );
      
      if (response.ok) {
        const data = await response.json();
        setProductReport(data);
      } else {
        alert('Error al generar el reporte');
      }
    } catch (error) {
      console.error('Error fetching product report:', error);
      alert('Error al generar el reporte');
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('es-ES', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    });
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold text-gray-800">Reportes de Ventas</h2>
        <p className="text-gray-600 mt-1">Analiza el desempeño de ventas y productos</p>
      </div>

      {/* Report Type Selector */}
      <div className="bg-white rounded-lg shadow-md p-4">
        <div className="flex space-x-4">
          <button
            onClick={() => setActiveReport('sales')}
            className={`px-4 py-2 rounded-lg font-medium transition ${
              activeReport === 'sales'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            Reporte de Ventas
          </button>
          <button
            onClick={() => setActiveReport('products')}
            className={`px-4 py-2 rounded-lg font-medium transition ${
              activeReport === 'products'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            Reporte por Producto
          </button>
        </div>
      </div>

      {/* Date Range Filter */}
      <div className="bg-white rounded-lg shadow-md p-4">
        <form
          onSubmit={activeReport === 'sales' ? fetchSalesReport : fetchProductReport}
          className="flex flex-wrap gap-4 items-end"
        >
          <div className="flex-1 min-w-[200px]">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Fecha Inicio
            </label>
            <input
              type="date"
              value={dateRange.start_date}
              onChange={(e) => setDateRange({ ...dateRange, start_date: e.target.value })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              required
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
              required
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? 'Generando...' : 'Generar Reporte'}
          </button>
        </form>
      </div>

      {/* Sales Report */}
      {activeReport === 'sales' && salesReport && (
        <div className="space-y-6">
          {/* Summary Cards */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="bg-white rounded-lg shadow-md p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-600 mb-1">Total Ventas</p>
                  <p className="text-3xl font-bold text-blue-600">
                    {salesReport.total_sales || 0}
                  </p>
                </div>
                <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
                  <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                  </svg>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-lg shadow-md p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-600 mb-1">Ingresos Brutos</p>
                  <p className="text-3xl font-bold text-green-600">
                    ${salesReport.gross_sales?.toFixed(2) || '0.00'}
                  </p>
                </div>
                <div className="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
                  <svg className="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-lg shadow-md p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-600 mb-1">Descuentos</p>
                  <p className="text-3xl font-bold text-red-600">
                    ${salesReport.total_discounts?.toFixed(2) || '0.00'}
                  </p>
                </div>
                <div className="w-12 h-12 bg-red-100 rounded-lg flex items-center justify-center">
                  <svg className="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
                  </svg>
                </div>
              </div>
            </div>
          </div>

          {/* Net Sales */}
          <div className="bg-gradient-to-r from-purple-500 to-blue-600 rounded-lg shadow-lg p-8 text-white">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-lg opacity-90 mb-2">Ingresos Netos</p>
                <p className="text-5xl font-bold">
                  ${salesReport.net_sales?.toFixed(2) || '0.00'}
                </p>
                <p className="text-sm opacity-75 mt-2">
                  Período: {formatDate(dateRange.start_date)} - {formatDate(dateRange.end_date)}
                </p>
              </div>
              <div className="w-20 h-20 bg-white bg-opacity-20 rounded-full flex items-center justify-center">
                <svg className="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
                </svg>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Product Report */}
      {activeReport === 'products' && productReport && (
        <div className="bg-white rounded-lg shadow-md overflow-hidden">
          <div className="p-6">
            <h3 className="text-lg font-bold text-gray-900 mb-4">
              Productos Más Vendidos
            </h3>
            <p className="text-sm text-gray-600 mb-6">
              Período: {formatDate(dateRange.start_date)} - {formatDate(dateRange.end_date)}
            </p>
          </div>

          {productReport.length === 0 ? (
            <div className="p-8 text-center">
              <p className="text-gray-600">No hay datos de ventas en este período</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      #
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Producto
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Cantidad Vendida
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Ingresos Totales
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Precio Promedio
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {productReport.map((product, index) => (
                    <tr key={product.product_id} className="hover:bg-gray-50">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="text-sm font-semibold text-gray-900">
                          {index + 1}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <span className="text-sm font-medium text-gray-900">
                          {product.product_name}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <span className="text-sm text-gray-900 font-semibold mr-2">
                            {product.total_quantity}
                          </span>
                          <span className="text-xs text-gray-500">unidades</span>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="text-sm font-bold text-green-600">
                          ${product.total_revenue?.toFixed(2) || '0.00'}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="text-sm text-gray-900">
                          ${(product.total_revenue / product.total_quantity).toFixed(2)}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
                <tfoot className="bg-gray-50">
                  <tr>
                    <td colSpan="2" className="px-6 py-4 text-sm font-bold text-gray-900">
                      TOTALES
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="text-sm font-bold text-gray-900">
                        {productReport.reduce((sum, p) => sum + p.total_quantity, 0)} unidades
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className="text-sm font-bold text-green-600">
                        ${productReport.reduce((sum, p) => sum + (p.total_revenue || 0), 0).toFixed(2)}
                      </span>
                    </td>
                    <td></td>
                  </tr>
                </tfoot>
              </table>
            </div>
          )}
        </div>
      )}

      {/* Empty State */}
      {((activeReport === 'sales' && !salesReport) || (activeReport === 'products' && !productReport)) && !loading && (
        <div className="bg-white rounded-lg shadow-md p-12 text-center">
          <svg className="w-16 h-16 text-gray-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          <h3 className="text-lg font-medium text-gray-900 mb-2">
            Selecciona un rango de fechas
          </h3>
          <p className="text-gray-600">
            Elige las fechas de inicio y fin para generar el reporte
          </p>
        </div>
      )}
    </div>
  );
}
