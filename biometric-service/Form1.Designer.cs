namespace BiometricPOC
{
    partial class Form1
    {
        private System.ComponentModel.IContainer components = null;

        protected override void Dispose(bool disposing)
        {
            if (disposing && (components != null)) components.Dispose();
            base.Dispose(disposing);
        }

        private void InitializeComponent()
        {
            this.txtLog = new System.Windows.Forms.TextBox();
            this.btnEnroll = new System.Windows.Forms.Button();
            this.btnVerify = new System.Windows.Forms.Button();
            this.btnClear = new System.Windows.Forms.Button();
            this.panel1 = new System.Windows.Forms.Panel();
            this.panel1.SuspendLayout();
            this.SuspendLayout();

            // txtLog
            this.txtLog.BackColor = System.Drawing.Color.FromArgb(20, 20, 20);
            this.txtLog.Dock = System.Windows.Forms.DockStyle.Fill;
            this.txtLog.Font = new System.Drawing.Font("Consolas", 10F);
            this.txtLog.ForeColor = System.Drawing.Color.LimeGreen;
            this.txtLog.Multiline = true;
            this.txtLog.ReadOnly = true;
            this.txtLog.ScrollBars = System.Windows.Forms.ScrollBars.Vertical;
            this.txtLog.Name = "txtLog";
            this.txtLog.TabIndex = 0;

            // btnEnroll
            this.btnEnroll.BackColor = System.Drawing.Color.FromArgb(34, 197, 94);
            this.btnEnroll.FlatStyle = System.Windows.Forms.FlatStyle.Flat;
            this.btnEnroll.Font = new System.Drawing.Font("Segoe UI", 11F, System.Drawing.FontStyle.Bold);
            this.btnEnroll.ForeColor = System.Drawing.Color.White;
            this.btnEnroll.Location = new System.Drawing.Point(10, 10);
            this.btnEnroll.Name = "btnEnroll";
            this.btnEnroll.Size = new System.Drawing.Size(200, 45);
            this.btnEnroll.Text = "🖐 Enroll (4 capturas)";
            this.btnEnroll.UseVisualStyleBackColor = false;
            this.btnEnroll.Click += new System.EventHandler(this.btnEnroll_Click);

            // btnVerify
            this.btnVerify.BackColor = System.Drawing.Color.FromArgb(59, 130, 246);
            this.btnVerify.FlatStyle = System.Windows.Forms.FlatStyle.Flat;
            this.btnVerify.Font = new System.Drawing.Font("Segoe UI", 11F, System.Drawing.FontStyle.Bold);
            this.btnVerify.ForeColor = System.Drawing.Color.White;
            this.btnVerify.Location = new System.Drawing.Point(220, 10);
            this.btnVerify.Name = "btnVerify";
            this.btnVerify.Size = new System.Drawing.Size(180, 45);
            this.btnVerify.Text = "🔍 Verificar";
            this.btnVerify.UseVisualStyleBackColor = false;
            this.btnVerify.Click += new System.EventHandler(this.btnVerify_Click);

            // btnClear
            this.btnClear.BackColor = System.Drawing.Color.FromArgb(100, 100, 100);
            this.btnClear.FlatStyle = System.Windows.Forms.FlatStyle.Flat;
            this.btnClear.Font = new System.Drawing.Font("Segoe UI", 11F);
            this.btnClear.ForeColor = System.Drawing.Color.White;
            this.btnClear.Location = new System.Drawing.Point(410, 10);
            this.btnClear.Name = "btnClear";
            this.btnClear.Size = new System.Drawing.Size(120, 45);
            this.btnClear.Text = "🗑 Limpiar";
            this.btnClear.UseVisualStyleBackColor = false;
            this.btnClear.Click += new System.EventHandler(this.btnClear_Click);

            // panel1 (botones)
            this.panel1.BackColor = System.Drawing.Color.FromArgb(30, 30, 30);
            this.panel1.Controls.Add(this.btnEnroll);
            this.panel1.Controls.Add(this.btnVerify);
            this.panel1.Controls.Add(this.btnClear);
            this.panel1.Dock = System.Windows.Forms.DockStyle.Bottom;
            this.panel1.Height = 65;
            this.panel1.Name = "panel1";

            // Form1
            this.AutoScaleDimensions = new System.Drawing.SizeF(7F, 15F);
            this.AutoScaleMode = System.Windows.Forms.AutoScaleMode.Font;
            this.BackColor = System.Drawing.Color.FromArgb(20, 20, 20);
            this.ClientSize = new System.Drawing.Size(750, 500);
            this.Controls.Add(this.txtLog);
            this.Controls.Add(this.panel1);
            this.Font = new System.Drawing.Font("Segoe UI", 9F);
            this.Name = "Form1";
            this.Text = "GymGo — Biometric POC";
            this.panel1.ResumeLayout(false);
            this.ResumeLayout(false);
            this.PerformLayout();
        }

        private System.Windows.Forms.TextBox txtLog;
        private System.Windows.Forms.Button btnEnroll;
        private System.Windows.Forms.Button btnVerify;
        private System.Windows.Forms.Button btnClear;
        private System.Windows.Forms.Panel panel1;
    }
}
