import Layout from '../../components/Layout'
export default function Settings(){
  return (
    <Layout>
      <div className="p-8">
        <h1 className="text-2xl font-bold mb-4">Settings</h1>
        <section className="mb-6"><h2 className="font-semibold mb-2">Header Style</h2><button>Horizontal Top Bar</button> <button>Vertical Sidebar</button></section>
        <section className="mb-6"><h2 className="font-semibold mb-2">Work Hours</h2><div>Working Days: Mon-Fri</div><div>Start: 9:00 | End: 17:00</div></section>
        <section className="mb-6"><h2 className="font-semibold mb-2">General</h2><select><option>Timezone</option></select></section>
        <section className="mb-6"><h2 className="font-semibold mb-2">Feature Toggles</h2>
          <label className="block"><input type="checkbox"/> Enable Dreams View</label>
          <label className="block"><input type="checkbox"/> Enable Goals View</label>
          <label className="block"><input type="checkbox"/> Enable Projects View</label>
          <label className="block"><input type="checkbox"/> Enable Opportunities View</label>
          <label className="block"><input type="checkbox"/> Enable Needs View</label>
          <label className="block"><input type="checkbox"/> Enable Graph View</label>
          <label className="block"><input type="checkbox"/> Enable Timeline View</label>
          <label className="block"><input type="checkbox"/> Enable Tools Page</label>
        </section>
        <section><h2 className="font-semibold mb-2">Data Management</h2><button>Export Data</button> <button>Import Data</button> <button>Clear All Data</button></section>
      </div>
    </Layout>
  )
}
