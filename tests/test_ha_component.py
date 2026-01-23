"""
Test HA component data parsing without full HA installation
"""
import asyncio

import aiohttp


async def test_coordinator():
    """Simulate HA coordinator fetch"""
    api_url = "http://localhost:8000"

    endpoints = {
        "generation": "/api/v1/generation-mix",
        "load": "/api/v1/load",
        "balance": "/api/v1/balance",
    }

    async with aiohttp.ClientSession() as session:
        data = {}

        for key, path in endpoints.items():
            url = f"{api_url}{path}"
            print(f"\n[TEST] Fetching: {url}")

            try:
                async with session.get(url, timeout=10) as response:
                    if response.status == 200:
                        data[key] = await response.json()

                        # Parse like HA sensor would
                        if "data" in data[key] and data[key]["data"]:
                            latest = data[key]["data"][0]
                            meta = data[key].get("meta", {})

                            print(f"  ✓ Status: {meta.get('quality_status')}")
                            print(f"  ✓ Source: {meta.get('source')}")
                            print(f"  ✓ Age: {meta.get('data_age_seconds')}s")

                            if key == "generation":
                                print(f"  ✓ Total: {latest.get('total_mw')} MW")
                            elif key == "load":
                                print(f"  ✓ Actual: {latest.get('actual_mw')} MW")
                            elif key == "balance":
                                print(f"  ✓ Delta: {latest.get('delta_mw')} MW")
                        else:
                            print("  ✗ No data in response")
                    else:
                        print(f"  ✗ HTTP {response.status}")
                        data[key] = None
            except Exception as e:
                print(f"  ✗ Error: {e}")
                data[key] = None

        return data

if __name__ == "__main__":
    print("="*60)
    print("TESTING SYNCTACLES HA COMPONENT DATA FETCH")
    print("="*60)

    result = asyncio.run(test_coordinator())

    print("\n" + "="*60)
    print("TEST RESULT:")
    print("="*60)

    for key in ["generation", "load", "balance"]:
        status = "✓ OK" if result.get(key) else "✗ FAIL"
        print(f"  {key:12} {status}")

    print("\n✅ Test complete")
