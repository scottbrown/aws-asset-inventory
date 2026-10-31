// Gives each resource type its own shape. Colour shows the service, so two types
// look the same only when one service has more types than there are shapes.
window.AssetShapes = (() => {
  'use strict';

  // Each shape has a 3D geometry (about 2 units across) and a 16x16 SVG glyph for the legend.
  const SHAPES = {
    sphere: {
      geometry: () => new THREE.SphereGeometry(1, 18, 12),
      svg: '<circle cx="8" cy="8" r="6"/>',
    },
    box: {
      geometry: () => new THREE.BoxGeometry(1.6, 1.6, 1.6),
      svg: '<rect x="2.5" y="2.5" width="11" height="11" rx="1"/>',
    },
    octahedron: {
      geometry: () => new THREE.OctahedronGeometry(1.3),
      svg: '<polygon points="8,1 15,8 8,15 1,8"/>',
    },
    tetrahedron: {
      geometry: () => new THREE.TetrahedronGeometry(1.5),
      svg: '<polygon points="8,1.5 14.5,14 1.5,14"/>',
    },
    cylinder: {
      geometry: () => new THREE.CylinderGeometry(0.9, 0.9, 1.7, 24),
      svg: '<path d="M3 4a5 2 0 0 1 10 0v8a5 2 0 0 1-10 0z"/><path d="M3 4a5 2 0 0 0 10 0" fill="none" stroke="#000" stroke-opacity=".45"/>',
    },
    bucket: {
      geometry: () => new THREE.CylinderGeometry(1.15, 0.7, 1.5, 24),
      svg: '<path d="M1.5 4a6.5 2 0 0 1 13 0l-2.5 10a4 1.5 0 0 1-8 0z"/><path d="M1.5 4a6.5 2 0 0 0 13 0" fill="none" stroke="#000" stroke-opacity=".45"/>',
    },
    cone: {
      geometry: () => new THREE.ConeGeometry(1, 1.9, 24),
      svg: '<path d="M8 1 13.5 12a5.5 2.5 0 0 1-11 0z"/><path d="M2.5 12a5.5 2.5 0 0 0 11 0" fill="none" stroke="#000" stroke-opacity=".45"/>',
    },
    torus: {
      geometry: () => new THREE.TorusGeometry(0.9, 0.38, 12, 32),
      svg: '<path fill-rule="evenodd" d="M8 2a6 6 0 1 0 0 12A6 6 0 1 0 8 2zm0 3.3a2.7 2.7 0 1 1 0 5.4 2.7 2.7 0 1 1 0-5.4z"/>',
    },
    dodecahedron: {
      geometry: () => new THREE.DodecahedronGeometry(1.15),
      svg: '<polygon points="8,1.5 14.5,6.2 12,14 4,14 1.5,6.2"/>',
    },
    icosahedron: {
      geometry: () => new THREE.IcosahedronGeometry(1.15),
      svg: '<polygon points="8,1.5 14,5 14,11 8,14.5 2,11 2,5"/>',
    },
    capsule: {
      geometry: () => new THREE.CapsuleGeometry(0.55, 1.0, 6, 16),
      svg: '<rect x="5" y="1.5" width="6" height="13" rx="3"/>',
    },
    slab: {
      geometry: () => new THREE.BoxGeometry(2, 0.5, 1.4),
      svg: '<rect x="1.5" y="5.5" width="13" height="5" rx="1"/>',
    },
    knot: {
      geometry: () => new THREE.TorusKnotGeometry(0.7, 0.22, 64, 10),
      svg: '<path d="M4.5 5a3 3 0 1 0 0 6c2 0 5-6 7-6a3 3 0 1 1 0 6c-2 0-5-6-7-6z" fill="none" stroke="currentColor" stroke-width="2"/>',
    },
  };

  // The order in which unlisted types of a service get the shapes that are still free.
  const SHAPE_ORDER = [
    'sphere', 'box', 'octahedron', 'cylinder', 'torus', 'dodecahedron', 'icosahedron',
    'capsule', 'slab', 'knot', 'cone', 'tetrahedron', 'bucket',
  ];

  // Shapes for common types, chosen to suggest what the resource is.
  // A container holds other resources and is drawn as an outline.
  const KNOWN_TYPES = {
    'AWS::EC2::Instance': { shape: 'box' },
    'AWS::EC2::Volume': { shape: 'cylinder' },
    'AWS::EC2::NetworkInterface': { shape: 'sphere' },
    'AWS::EC2::SecurityGroup': { shape: 'torus' },
    'AWS::EC2::EIP': { shape: 'capsule' },
    'AWS::EC2::InternetGateway': { shape: 'cone' },
    'AWS::EC2::NatGateway': { shape: 'octahedron' },
    'AWS::EC2::RouteTable': { shape: 'slab' },
    'AWS::EC2::NetworkAcl': { shape: 'dodecahedron' },
    'AWS::EC2::VPCEndpoint': { shape: 'knot' },
    'AWS::EC2::VPC': { shape: 'icosahedron', container: true },
    'AWS::EC2::Subnet': { shape: 'box', container: true },
    'AWS::EC2::TransitGateway': { shape: 'sphere', container: true },
    'AWS::S3::Bucket': { shape: 'bucket' },
    'AWS::Lambda::Function': { shape: 'octahedron' },
    'AWS::IAM::Role': { shape: 'icosahedron' },
    'AWS::IAM::User': { shape: 'capsule' },
    'AWS::IAM::Group': { shape: 'dodecahedron' },
    'AWS::IAM::Policy': { shape: 'slab' },
    'AWS::RDS::DBInstance': { shape: 'cylinder' },
    'AWS::RDS::DBCluster': { shape: 'cylinder', container: true },
    'AWS::RDS::DBSubnetGroup': { shape: 'box', container: true },
    'AWS::DynamoDB::Table': { shape: 'cylinder' },
    'AWS::ElasticLoadBalancingV2::LoadBalancer': { shape: 'cone' },
    'AWS::ElasticLoadBalancing::LoadBalancer': { shape: 'cone' },
    'AWS::AutoScaling::AutoScalingGroup': { shape: 'dodecahedron', container: true },
    'AWS::ECS::Cluster': { shape: 'icosahedron', container: true },
    'AWS::ECS::Service': { shape: 'dodecahedron' },
    'AWS::ECS::TaskDefinition': { shape: 'slab' },
    'AWS::EKS::Cluster': { shape: 'icosahedron', container: true },
    'AWS::CloudFormation::Stack': { shape: 'box', container: true },
    'AWS::KMS::Key': { shape: 'knot' },
    'AWS::SNS::Topic': { shape: 'cone' },
    'AWS::SQS::Queue': { shape: 'capsule' },
    'AWS::CloudTrail::Trail': { shape: 'slab' },
    'AWS::CloudWatch::Alarm': { shape: 'tetrahedron' },
  };

  const geometries = new Map();

  function geometry(shape) {
    if (!geometries.has(shape)) geometries.set(shape, SHAPES[shape].geometry());
    return geometries.get(shape);
  }

  function serviceOf(type) {
    const parts = type.split('::');
    return parts.length >= 2 ? parts[1] : type;
  }

  // Returns a Map of type -> { shape, container }. Within a service, each type
  // gets a different shape until the shapes run out.
  function assign(types) {
    const styles = new Map();
    const used = new Map(); // service -> Set of shapes
    const take = (type, style) => {
      styles.set(type, style);
      const service = serviceOf(type);
      if (!used.has(service)) used.set(service, new Set());
      used.get(service).add(style.shape + (style.container ? '+' : ''));
    };

    const sorted = [...new Set(types)].sort();
    for (const type of sorted) {
      if (KNOWN_TYPES[type]) take(type, KNOWN_TYPES[type]);
    }
    for (const type of sorted) {
      if (styles.has(type)) continue;
      const taken = used.get(serviceOf(type)) || new Set();
      const free = SHAPE_ORDER.find((s) => !taken.has(s));
      take(type, { shape: free || SHAPE_ORDER[hash(type) % SHAPE_ORDER.length], container: false });
    }
    return styles;
  }

  function hash(s) {
    let h = 0;
    for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
    return h;
  }

  // Creates the 3D object for one node. Call paint() to change its colour.
  function create(style, radius) {
    const g = geometry(style.shape);
    let obj;
    if (style.container) {
      obj = new THREE.Mesh(g, new THREE.MeshLambertMaterial({ transparent: true, opacity: 0.12, depthWrite: false }));
      obj.add(new THREE.LineSegments(new THREE.EdgesGeometry(g, 20), new THREE.LineBasicMaterial()));
    } else {
      obj = new THREE.Mesh(g, new THREE.MeshLambertMaterial());
    }
    obj.scale.setScalar(radius);
    return obj;
  }

  function paint(obj, color) {
    obj.material.color.set(color);
    for (const child of obj.children) child.material.color.set(color);
  }

  // Returns an SVG element that shows the shape in the legend.
  function glyph(style, color) {
    const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    svg.setAttribute('viewBox', '0 0 16 16');
    svg.setAttribute('class', 'glyph');
    svg.setAttribute('aria-hidden', 'true');
    svg.style.color = color;
    if (style.container) {
      svg.setAttribute('fill', 'currentColor');
      svg.setAttribute('fill-opacity', '0.15');
      svg.setAttribute('stroke', 'currentColor');
      svg.setAttribute('stroke-width', '1.4');
    } else {
      svg.setAttribute('fill', 'currentColor');
    }
    svg.innerHTML = SHAPES[style.shape].svg; // constant markup, no user data
    return svg;
  }

  return { assign, create, paint, glyph, serviceOf };
})();
